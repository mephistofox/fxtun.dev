---
title: "CI/CD и туннели: preview для каждого pull request"
date: 2026-03-05T14:00:00+03:00
draft: false
description: "Настройте автоматический preview-URL для каждого pull request через GitHub Actions и fxTunnel — ревьюеры получают рабочую ссылку вместо ветки."
tags: [ci-cd, preview, deployment, fxTunnel, github-actions, developer-tools]
image: ""
rubrics: [guides]
need: tunnels
cover: ["pull request", "fxTunnel CI", "preview-URL"]
updated: 2026-09-27
---

## Проблема: ревью PR без живого превью -- это медленно

Вы открываете diff в GitHub и видите... код. А как приложение реально себя ведёт -- не видите. Чтобы проверить PR вживую, нужно переключить ветку, поставить зависимости, запустить проект и прокликать сценарии руками. Это 10-15 минут на каждый PR, и большинство ревьюеров просто пропускают этот шаг.

Результат предсказуем: баги проскакивают в main, QA ловит проблемы уже после мержа, а исправления превращаются в новые PR. Цикл обратной связи растягивается с минут до часов. [Локальный туннель](/blog/what-is-tunneling/) и CI/CD могут это исправить: каждый PR получает свой публичный URL с работающим приложением.

## Традиционные подходы к preview-окружениям

Preview-окружения -- не новая идея. Vercel и Netlify давно умеют разворачивать preview для каждого PR. Но у обоих решений общее ограничение: они работают только с фронтендом.

### Vercel / Netlify: только фронтенд

Vercel разворачивает preview для Next.js, Astro и других фреймворков с SSR или статической генерацией. Netlify делает то же для статических сайтов и serverless-функций. Оба сервиса отлично подходят для фронтенд-проектов.

Но если ваше приложение — это Go-бэкенд с PostgreSQL, Python API с Celery и Redis, или монолит на Java — Vercel вам не поможет. Вам нужно preview для полного стека, а не только для фронтенда.

### Staging-сервер: один на всех

Классический staging-сервер — это общее окружение, куда деплоятся изменения для тестирования. Проблема в том, что staging один, а PR — много. Два разработчика деплоят одновременно — и чьи-то изменения перезаписывают чужие. Очередь на staging тормозит всю команду.

### Kubernetes preview namespaces: мощно, но сложно

Некоторые команды создают отдельный namespace в Kubernetes для каждого PR. Это работает, но требует серьёзной инфраструктуры: Helm-чарты, Ingress-контроллеры, wildcard DNS, автоочистка. Для команды из 5-10 человек это избыточно.

## Full-stack preview с туннелями: идея

Подход прост. CI-раннер собирает и запускает приложение, fxTunnel создаёт [публичный URL](/blog/expose-localhost-to-internet/) к нему, а бот комментирует PR ссылкой. Ревьюер кликает -- и видит работающее приложение.

{{< flow caption="fxTunnel на CI-раннере даёт приложению публичный URL; когда PR закрывается, туннель останавливается вместе с джобом." >}}
- {label: ревьюер, title: браузер или E2E-тест, sub: клик по ссылке из PR}
- {link: HTTPS, kind: tls}
- {label: fxTunnel Server, title: relay, sub: публичный URL}
- {link: TLS, kind: tls}
- {label: CI-раннер, title: клиент fxtunnel, sub: "docker compose up · fxtunnel http 8080", me: true}
{{< /flow >}}

Преимущества очевидны: preview работает с любым стеком, не нужна отдельная инфраструктура, а туннель автоматически закрывается, когда CI-джоб завершается.

## GitHub Actions: полный workflow

Ниже — готовый workflow для GitHub Actions. Он собирает приложение через Docker Compose, запускает fxTunnel и комментирует PR публичным URL. Вы можете скопировать его и адаптировать под свой проект.

### Основной workflow

```yaml
# .github/workflows/preview.yml
name: PR Preview Environment

on:
  pull_request:
    types: [opened, synchronize, reopened]

concurrency:
  group: preview-${{ github.event.pull_request.number }}
  cancel-in-progress: true

jobs:
  preview:
    runs-on: ubuntu-latest
    timeout-minutes: 30
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Start application
        run: |
          docker compose -f docker-compose.preview.yml up -d --build
          # Ждём, пока приложение станет доступным
          for i in $(seq 1 30); do
            curl -sf http://localhost:8080/health && break
            echo "Waiting for app to start... ($i/30)"
            sleep 2
          done

      - name: Install fxTunnel
        run: curl -fsSL https://fxtun.ru/install.sh | bash

      - name: Start tunnel and get URL
        id: tunnel
        run: |
          # Запускаем туннель в фоне и перехватываем URL
          fxtunnel http 8080 > /tmp/tunnel.log 2>&1 &
          TUNNEL_PID=$!
          echo "tunnel_pid=$TUNNEL_PID" >> "$GITHUB_OUTPUT"

          # Ждём, пока URL появится в логах
          for i in $(seq 1 15); do
            TUNNEL_URL=$(grep -oP 'https://[a-z0-9-]+\.fxtun\.ru' /tmp/tunnel.log || true)
            if [ -n "$TUNNEL_URL" ]; then
              echo "preview_url=$TUNNEL_URL" >> "$GITHUB_OUTPUT"
              echo "Preview URL: $TUNNEL_URL"
              break
            fi
            sleep 1
          done

          if [ -z "$TUNNEL_URL" ]; then
            echo "Failed to get tunnel URL"
            exit 1
          fi

      - name: Comment PR with preview URL
        uses: actions/github-script@v7
        with:
          script: |
            const previewUrl = '${{ steps.tunnel.outputs.preview_url }}';
            const body = `## Preview Environment

            | | |
            |---|---|
            | **URL** | ${previewUrl} |
            | **Commit** | \`${context.sha.substring(0, 7)}\` |
            | **Status** | Active |

            This preview will be available while the CI job is running.`;

            // Удаляем старый комментарий, если есть
            const comments = await github.rest.issues.listComments({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: context.issue.number,
            });
            const botComment = comments.data.find(c =>
              c.body.includes('## Preview Environment')
            );
            if (botComment) {
              await github.rest.issues.deleteComment({
                owner: context.repo.owner,
                repo: context.repo.repo,
                comment_id: botComment.id,
              });
            }

            await github.rest.issues.createComment({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: context.issue.number,
              body: body,
            });

      - name: Run E2E tests against preview
        run: |
          npx playwright test --base-url=${{ steps.tunnel.outputs.preview_url }}

      - name: Keep preview alive for manual testing
        if: success()
        run: |
          echo "Preview is live at: ${{ steps.tunnel.outputs.preview_url }}"
          echo "Keeping alive for 20 minutes for manual review..."
          sleep 1200

      - name: Cleanup
        if: always()
        run: |
          kill ${{ steps.tunnel.outputs.tunnel_pid }} || true
          docker compose -f docker-compose.preview.yml down
```

### Docker Compose для preview

```yaml
# docker-compose.preview.yml
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://user:pass@db:5432/preview
      - REDIS_URL=redis://redis:6379
      - APP_ENV=preview
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_started

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
      POSTGRES_DB: preview
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user"]
      interval: 2s
      timeout: 5s
      retries: 10

  redis:
    image: redis:7-alpine
```

Подробнее об использовании Docker с туннелями — в статье [Docker + туннель](/blog/docker-tunnel-expose-containers/).

## GitLab CI: краткий пример

Аналогичный workflow для GitLab CI. Принцип тот же: запуск приложения, создание туннеля, комментарий к merge request.

```yaml
# .gitlab-ci.yml
preview:
  stage: test
  image: docker:24
  services:
    - docker:24-dind
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
  script:
    - docker compose -f docker-compose.preview.yml up -d --build
    - curl -fsSL https://fxtun.ru/install.sh | bash
    - fxtunnel http 8080 > /tmp/tunnel.log 2>&1 &
    - sleep 5
    - TUNNEL_URL=$(grep -oP 'https://[a-z0-9-]+\.fxtun\.ru' /tmp/tunnel.log)
    - |
      curl --request POST \
        --header "PRIVATE-TOKEN: ${GITLAB_TOKEN}" \
        --header "Content-Type: application/json" \
        --data "{\"body\": \"Preview: ${TUNNEL_URL}\"}" \
        "${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/merge_requests/${CI_MERGE_REQUEST_IID}/notes"
    - echo "Preview live at ${TUNNEL_URL}"
    - sleep 1200
  after_script:
    - docker compose -f docker-compose.preview.yml down
```

## Что даёт tunnel preview команде

Что меняется, когда к каждому PR прикладывается живая ссылка? На удивление многое.

### Тестирование вебхуков прямо из PR

Если ваше приложение принимает [вебхуки от Stripe, GitHub или Telegram](/blog/webhook-testing-with-tunnel/), preview-окружение позволяет настроить тестовый вебхук на URL из PR и проверить интеграцию до мержа. QA-инженер может протригерить платёж в тестовом режиме Stripe и увидеть, как приложение его обрабатывает.

### Демонстрация для QA и дизайнеров

Ссылка в комментарии к PR позволяет любому члену команды открыть preview в браузере — без клонирования репозитория, установки зависимостей и запуска проекта локально. Дизайнер проверяет UI, продакт-менеджер — бизнес-логику, QA — edge cases.

### E2E-тесты против реального URL

Публичный HTTPS-URL от fxTunnel можно передать в Playwright, Cypress или Selenium. Тесты выполняются против приложения, работающего в условиях, приближённых к продакшену: реальный HTTP, TLS, DNS-резолвинг. Это надёжнее, чем тесты против `localhost`.

### Тестирование мобильных приложений

Если к бэкенду подключается мобильное приложение, QA может вписать URL из PR в конфигурацию тестовой сборки и проверить интеграцию на реальном устройстве.

## Сравнение стоимости: tunnel preview vs альтернативы

| Подход | Стоимость | Полный стек | Изоляция PR | Настройка |
|---|---|---|---|---|
| **fxTunnel** | Бесплатно (1 PR), платно для нескольких параллельно | Да | Да | 10 минут |
| **Vercel preview** | $0-20/мес | Только фронтенд | Да | 5 минут |
| **Netlify preview** | $0-19/мес | Только фронтенд | Да | 5 минут |
| **Staging-сервер** | $20-100/мес | Да | Нет (один на всех) | Часы |
| **K8s per-PR namespaces** | $50-200/мес | Да | Да | Дни |

Для full-stack проектов fxTunnel даёт изолированное preview для каждого PR с поддержкой любого стека. На бесплатном тарифе одновременно доступен только 1 туннель, так что для нескольких открытых PR параллельно нужен тариф Base и выше (5 туннелей) — платные тарифы также добавляют кастомный домен для preview-URL. Vercel и Netlify остаются отличным выбором для чисто фронтенд-проектов. Сравнение инструментов -- в статье [ngrok vs Cloudflare vs fxTunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

## Автоочистка: закрытие туннеля при мерже или закрытии PR

Туннель, запущенный в CI-джобе, автоматически закрывается при завершении джоба. Но если вы используете `sleep` для поддержания preview активным, нужен механизм отмены при мерже или закрытии PR.

GitHub Actions решает это через `concurrency` и отдельный workflow на закрытие.

### Workflow для отмены при закрытии PR

```yaml
# .github/workflows/preview-cleanup.yml
name: Cleanup Preview

on:
  pull_request:
    types: [closed]

jobs:
  cleanup:
    runs-on: ubuntu-latest
    steps:
      - name: Cancel preview workflow
        uses: actions/github-script@v7
        with:
          script: |
            // Находим активные workflow runs для этого PR
            const runs = await github.rest.actions.listWorkflowRuns({
              owner: context.repo.owner,
              repo: context.repo.repo,
              workflow_id: 'preview.yml',
              status: 'in_progress',
            });

            for (const run of runs.data.workflow_runs) {
              // Проверяем, что run относится к нашему PR
              if (run.pull_requests.some(pr => pr.number === context.issue.number)) {
                await github.rest.actions.cancelWorkflowRun({
                  owner: context.repo.owner,
                  repo: context.repo.repo,
                  run_id: run.id,
                });
                console.log(`Cancelled workflow run ${run.id}`);
              }
            }

      - name: Update PR comment
        uses: actions/github-script@v7
        with:
          script: |
            const comments = await github.rest.issues.listComments({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: context.issue.number,
            });
            const botComment = comments.data.find(c =>
              c.body.includes('## Preview Environment')
            );
            if (botComment) {
              await github.rest.issues.updateComment({
                owner: context.repo.owner,
                repo: context.repo.repo,
                comment_id: botComment.id,
                body: `## Preview Environment\n\n**Status:** Closed (PR ${context.payload.action})`,
              });
            }
```

Ключевой момент — секция `concurrency` в основном workflow. Параметр `cancel-in-progress: true` гарантирует, что при новом коммите в PR старый preview-джоб отменяется, а новый запускается с обновлённым кодом. При закрытии PR отдельный workflow отменяет активный preview.

## Советы по настройке

### Кэширование Docker-слоёв

Сборка Docker-образов при каждом push может быть медленной. Используйте кэш:

```yaml
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Cache Docker layers
        uses: actions/cache@v4
        with:
          path: /tmp/.buildx-cache
          key: ${{ runner.os }}-buildx-${{ github.sha }}
          restore-keys: ${{ runner.os }}-buildx-
```

### Health check перед открытием туннеля

Не запускайте туннель, пока приложение не готово. Используйте health check endpoint:

```yaml
      - name: Wait for application
        run: |
          for i in $(seq 1 30); do
            if curl -sf http://localhost:8080/health; then
              echo "Application is ready"
              exit 0
            fi
            echo "Waiting... ($i/30)"
            sleep 2
          done
          echo "Application failed to start"
          exit 1
```

### Ограничение времени жизни preview

Устанавливайте разумный `timeout-minutes` для джоба и `sleep` для ожидания. 20-30 минут обычно достаточно для ручного тестирования. Для автоматических E2E-тестов preview может жить только во время прогона тестов.

