<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useThemeStore, type ThemeMode } from '@/stores/theme'
import { setLocale, getLocale } from '@/i18n'
import { useSeo } from '@/composables/useSeo'

const themeStore = useThemeStore()
const { t, locale } = useI18n()

useSeo({ titleKey: 'seo.terms.title', descriptionKey: 'seo.terms.description' })

function toggleLocale() {
  const current = getLocale()
  setLocale(current === 'en' ? 'ru' : 'en')
}

function cycleTheme() {
  const modes: ThemeMode[] = ['light', 'dark', 'system']
  const currentIndex = modes.indexOf(themeStore.mode)
  const nextIndex = (currentIndex + 1) % modes.length
  themeStore.setMode(modes[nextIndex])
}

const lastUpdated = '13.02.2026'
</script>

<template>
  <div class="min-h-screen bg-background">
    <!-- Theme and Language Switchers -->
    <div class="fixed top-4 right-4 flex items-center space-x-2 z-50">
      <button
        @click="cycleTheme"
        class="p-2 rounded-lg hover:bg-accent/10 transition-colors"
        :title="t(`theme.${themeStore.mode}`)"
      >
        <svg
          v-if="themeStore.mode === 'light'"
          xmlns="http://www.w3.org/2000/svg"
          class="h-5 w-5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <circle cx="12" cy="12" r="5" />
          <line x1="12" y1="1" x2="12" y2="3" />
          <line x1="12" y1="21" x2="12" y2="23" />
          <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
          <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
          <line x1="1" y1="12" x2="3" y2="12" />
          <line x1="21" y1="12" x2="23" y2="12" />
          <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
          <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
        </svg>
        <svg
          v-else-if="themeStore.mode === 'dark'"
          xmlns="http://www.w3.org/2000/svg"
          class="h-5 w-5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
        </svg>
        <svg
          v-else
          xmlns="http://www.w3.org/2000/svg"
          class="h-5 w-5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
          <line x1="8" y1="21" x2="16" y2="21" />
          <line x1="12" y1="17" x2="12" y2="21" />
        </svg>
      </button>
      <button
        @click="toggleLocale"
        class="px-2 py-1 text-sm font-medium rounded-lg hover:bg-accent/10 transition-colors"
      >
        {{ getLocale() === 'en' ? 'RU' : 'EN' }}
      </button>
    </div>

    <!-- Back to landing -->
    <RouterLink
      to="/"
      class="fixed top-4 left-4 flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors z-50"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M9.707 16.707a1 1 0 01-1.414 0l-6-6a1 1 0 010-1.414l6-6a1 1 0 011.414 1.414L5.414 9H17a1 1 0 110 2H5.414l4.293 4.293a1 1 0 010 1.414z" clip-rule="evenodd" />
      </svg>
      {{ t('landing.nav.backToHome') }}
    </RouterLink>

    <div class="container mx-auto px-4 py-16 max-w-4xl">
      <!-- Header -->
      <div class="mb-8">
        <h1 class="text-3xl font-bold mb-4">{{ t('legal.termsTitle') }}</h1>
        <span class="text-sm text-muted-foreground">
          {{ t('legal.lastUpdated') }}: {{ lastUpdated }}
        </span>
      </div>

      <!-- Content -->
      <div class="prose prose-neutral dark:prose-invert max-w-none">
        <template v-if="locale === 'ru'">
          <h2>1. Введение</h2>
          <p>
            Настоящие Условия использования («Условия») регулируют доступ и использование сервиса fxTunnel
            («Сервис»), включая веб-сайт <a href="https://fxtun.ru">fxtun.ru</a>,
            десктопные приложения, инструменты командной строки и все связанные API.
          </p>
          <p>
            Сервис управляется <strong>ИП Наводнюк А.И.</strong>
            (далее «Компания», «мы», «нас» или «наш»).
            Сайт: <a href="https://fxtun.ru">fxtun.ru</a>.
          </p>
          <p>
            Создавая аккаунт или используя Сервис, вы («Пользователь») соглашаетесь
            с настоящими Условиями. Если вы не согласны, не используйте Сервис.
          </p>

          <h2>2. Определения</h2>
          <ul>
            <li><strong>«Туннель»</strong> — защищённое соединение, предоставляющее доступ к локальному сервису через интернет посредством инфраструктуры fxTunnel.</li>
            <li><strong>«Субдомен»</strong> — уникальное имя хоста (например, <code>your-app.fxtun.ru</code>), назначенное HTTP-туннелю.</li>
            <li><strong>«Токен»</strong> — учётные данные API для аутентификации туннельных подключений.</li>
            <li><strong>«Тариф»</strong> — план подписки, определяющий объём доступных функций Сервиса (Free, Base или Pro).</li>
            <li><strong>«Инспектор трафика»</strong> — встроенный инструмент для мониторинга HTTP-запросов и ответов, проходящих через туннели.</li>
          </ul>

          <h2>3. Описание Сервиса</h2>
          <p>
            fxTunnel предоставляет безопасное обратное туннелирование, позволяющее открыть доступ
            к локальным HTTP, TCP и UDP сервисам через интернет. Сервис включает:
          </p>
          <ul>
            <li>HTTP-туннелирование с пользовательскими или случайными субдоменами в зоне <code>fxtun.ru</code>;</li>
            <li>Проброс TCP-портов с динамически выделяемыми публичными портами;</li>
            <li>Проброс UDP-портов;</li>
            <li>Веб-панель для управления туннелями, токенами и зарезервированными субдоменами;</li>
            <li>Десктопный GUI-клиент для Windows, macOS и Linux;</li>
            <li>CLI-клиент для всех основных платформ;</li>
            <li>Инспектор трафика в реальном времени для HTTP-туннелей.</li>
          </ul>

          <h2>4. Регистрация аккаунта</h2>
          <p>
            Для использования Сервиса необходимо создать аккаунт. Вы обязуетесь предоставить
            достоверную и полную информацию при регистрации и обеспечить безопасность учётных данных.
          </p>
          <p>
            Вы несёте полную ответственность за все действия, совершённые через ваш аккаунт.
            При подозрении на несанкционированный доступ немедленно свяжитесь с нами:
            <a href="mailto:support@fxtun.ru">support@fxtun.ru</a>.
          </p>

          <h2>5. Бесплатный и платные тарифы</h2>
          <h3>5.1. Бесплатный тариф</h3>
          <p>
            Бесплатный тариф предоставляет до 3 одновременных туннелей с любым доступным субдоменом,
            без лимитов трафика и без таймаута сессий. Бесплатный тариф действует бессрочно
            и не требует привязки карты.
          </p>
          <h3>5.2. Платные тарифы</h3>
          <p>
            Платные тарифы (Base и Pro) предоставляют дополнительные возможности: больше туннелей,
            зарезервированные субдомены, свои домены и приоритетные функции. Актуальные цены доступны на
            <a href="https://fxtun.ru/#pricing">fxtun.ru/#pricing</a>.
          </p>
          <p>
            Подписки оплачиваются ежемесячно. Оплата обрабатывается через сторонних платёжных провайдеров.
            Оформляя подписку, вы разрешаете автоматическое списание до момента отмены.
          </p>

          <h2>6. Допустимое использование</h2>
          <p>Запрещается использовать Сервис для:</p>
          <ul>
            <li>Нарушения применимого законодательства, нормативных актов или прав третьих лиц;</li>
            <li>Распространения вредоносного ПО, фишинговых страниц или иного вредоносного контента;</li>
            <li>Проведения DDoS-атак или сканирования портов третьих лиц;</li>
            <li>Размещения или распространения незаконного контента;</li>
            <li>Обхода средств контроля доступа или аутентификации других систем;</li>
            <li>Рассылки спама или нежелательных массовых сообщений;</li>
            <li>Вмешательства в работу Сервиса или его инфраструктуры;</li>
            <li>Перепродажи, перераспределения или сублицензирования доступа к Сервису без письменного согласия.</li>
          </ul>
          <p>
            Мы оставляем за собой право немедленно приостановить или прекратить действие вашего аккаунта
            без предварительного уведомления при нарушении настоящих Условий.
          </p>

          <h2>7. Интеллектуальная собственность</h2>
          <p>
            Сервис, включая программное обеспечение, дизайн, логотипы, документацию и всю связанную
            интеллектуальную собственность, принадлежит ИП Наводнюк А.И. или его лицензиарам. Настоящие Условия
            не предоставляют вам никаких прав на Сервис, кроме ограниченного права использования.
          </p>
          <p>
            Компоненты с открытым исходным кодом fxTunnel лицензируются в соответствии
            с их лицензиями, указанными в репозиториях исходного кода.
          </p>

          <h2>8. Конфиденциальность и обработка данных</h2>
          <h3>8.1. Какие данные мы собираем</h3>
          <ul>
            <li><strong>Данные аккаунта:</strong> адрес электронной почты, хэш пароля, настройки;</li>
            <li><strong>Метаданные подключений:</strong> IP-адреса, user agent, временные метки сессий туннелей;</li>
            <li><strong>Данные использования:</strong> количество туннелей, статистика трафика, использование функций;</li>
            <li><strong>Платёжные данные:</strong> обрабатываются и хранятся исключительно платёжными провайдерами — мы не храним данные карт.</li>
          </ul>
          <h3>8.2. Как мы используем данные</h3>
          <ul>
            <li>Для предоставления и поддержки Сервиса;</li>
            <li>Для аутентификации и управления аккаунтом;</li>
            <li>Для обработки платежей и управления подписками;</li>
            <li>Для обнаружения и предотвращения злоупотреблений;</li>
            <li>Для отправки важных уведомлений о Сервисе.</li>
          </ul>
          <h3>8.3. Хранение данных</h3>
          <p>
            Данные аккаунта хранятся в течение срока действия аккаунта плюс 12 месяцев после удаления.
            Метаданные подключений хранятся до 90 дней. Вы можете запросить полное удаление данных,
            обратившись по адресу <a href="mailto:support@fxtun.ru">support@fxtun.ru</a>.
          </p>
          <h3>8.4. Содержимое трафика</h3>
          <p>
            Мы <strong>не</strong> просматриваем, не записываем и не храним содержимое трафика,
            проходящего через туннели. Инспектор трафика работает исключительно в вашем браузере
            и на вашем устройстве — данные туннельного трафика не хранятся на наших серверах.
          </p>
          <h3>8.5. Передача данных третьим лицам</h3>
          <p>Мы можем передавать данные следующим категориям обработчиков:</p>
          <ul>
            <li>Платёжные провайдеры (для выставления счетов и управления подписками);</li>
            <li>Провайдеры инфраструктуры (для хостинга и доставки контента);</li>
            <li>Правоохранительные органы (по требованию применимого законодательства или решению суда).</li>
          </ul>
          <h3>8.6. Файлы cookie</h3>
          <p>
            Сервис использует строго необходимые cookie для аутентификации и управления сессиями.
            Мы не используем cookie для отслеживания или рекламы.
          </p>

          <h2>9. Отмена подписки и возвраты</h2>
          <p>
            Вы можете отменить подписку в любое время через панель управления аккаунтом.
            После отмены доступ к платным функциям сохраняется до конца оплаченного периода.
          </p>
          <ul>
            <li><strong>В течение 7 дней после первого платежа:</strong> полный возврат без вопросов.</li>
            <li><strong>После 7 дней:</strong> пропорциональный возврат за оставшиеся полные дни периода.</li>
          </ul>
          <p>
            Возвраты осуществляются на исходный способ оплаты в течение 14 рабочих дней.
            Для оформления возврата обратитесь по адресу <a href="mailto:support@fxtun.ru">support@fxtun.ru</a>.
          </p>

          <h2>10. Доступность Сервиса и SLA</h2>
          <p>
            Мы прилагаем коммерчески обоснованные усилия для поддержания доступности инфраструктуры
            туннелирования на уровне 99,9%. Однако Сервис предоставляется «как есть» и «по мере доступности».
          </p>
          <p>Мы не несём ответственности за перебои, вызванные:</p>
          <ul>
            <li>Плановым обслуживанием (уведомление не менее чем за 24 часа);</li>
            <li>Обстоятельствами непреодолимой силы: стихийные бедствия, войны, пандемии, действия властей;</li>
            <li>Сбоями сторонних сервисов (DNS-провайдеры, хостинг-провайдеры, платёжные системы);</li>
            <li>Проблемами с вашим сетевым подключением или локальным окружением.</li>
          </ul>

          <h2>11. Ограничение ответственности</h2>
          <p>
            В максимальной степени, допускаемой применимым законодательством, ИП Наводнюк А.И. не несёт
            ответственности за любые косвенные, случайные, специальные, штрафные убытки, включая,
            но не ограничиваясь, упущенную выгоду, потерю данных, деловых возможностей или репутации,
            возникшие в связи с использованием Сервиса.
          </p>
          <p>
            Совокупная ответственность по любым претензиям, связанным с Сервисом,
            не может превышать сумму, уплаченную вами за 12 месяцев, предшествующих претензии.
          </p>

          <h2>12. Возмещение убытков</h2>
          <p>
            Вы обязуетесь возместить и оградить ИП Наводнюк А.И. от любых претензий,
            убытков, потерь, обязательств и расходов
            (включая разумные судебные издержки), возникших из-за использования вами Сервиса
            или нарушения настоящих Условий.
          </p>

          <h2>13. Изменение Условий</h2>
          <p>
            Мы можем обновлять настоящие Условия. О существенных изменениях мы уведомим
            по электронной почте или заметным уведомлением в Сервисе не менее чем за 30 дней
            до вступления в силу. Продолжение использования Сервиса после даты вступления
            в силу означает принятие обновлённых Условий.
          </p>

          <h2>14. Прекращение действия</h2>
          <p>
            Любая из сторон может расторгнуть соглашение в любое время. Вы можете удалить свой аккаунт
            или обратиться в службу поддержки. Мы можем немедленно прекратить или приостановить
            ваш доступ за нарушение настоящих Условий без предварительного уведомления.
          </p>
          <p>
            После прекращения ваше право на использование Сервиса утрачивается немедленно.
            Положения об интеллектуальной собственности, ограничении ответственности, возмещении
            убытков и применимом праве сохраняют силу после прекращения.
          </p>

          <h2>15. Применимое право и разрешение споров</h2>
          <p>
            Настоящие Условия регулируются и толкуются в соответствии с законодательством
            Российской Федерации.
          </p>
          <p>
            Любые споры, возникающие из настоящих Условий или Сервиса, должны сначала разрешаться
            путём добросовестных переговоров. Если спор не разрешён в течение 30 дней,
            он передаётся на рассмотрение в суды Российской Федерации.
          </p>

          <h2>16. Делимость положений</h2>
          <p>
            Если какое-либо положение настоящих Условий будет признано недействительным
            или неисполнимым, остальные положения сохраняют полную юридическую силу.
          </p>

          <h2>17. Контактная информация</h2>
          <table class="w-full">
            <tbody>
              <tr>
                <td class="font-medium pr-4 py-1">Компания:</td>
                <td>ИП Наводнюк А.И.</td>
              </tr>
              <tr>
                <td class="font-medium pr-4 py-1">Юрисдикция:</td>
                <td>Российская Федерация</td>
              </tr>
              <tr>
                <td class="font-medium pr-4 py-1">Сайт:</td>
                <td><a href="https://fxtun.ru">fxtun.ru</a></td>
              </tr>
              <tr>
                <td class="font-medium pr-4 py-1">Email:</td>
                <td><a href="mailto:support@fxtun.ru">support@fxtun.ru</a></td>
              </tr>
            </tbody>
          </table>
        </template>

        <template v-else>
        <h2>1. Introduction</h2>
        <p>
          These Terms of Service ("Terms") govern your access to and use of the fxtun service
          ("Service"), including the website at <a href="https://fxtun.dev">fxtun.dev</a>,
          desktop applications, command-line tools, and all related APIs.
        </p>
        <p>
          The Service is operated by <strong>Nocodo LTD</strong>, a company incorporated and
          registered in the Republic of Cyprus (hereinafter "Company", "we", "us", or "our").
          Company website: <a href="https://nocodo.tech">nocodo.tech</a>.
        </p>
        <p>
          By creating an account or using the Service, you ("User", "you", or "your") agree to
          be bound by these Terms. If you do not agree, do not use the Service.
        </p>

        <h2>2. Definitions</h2>
        <ul>
          <li><strong>"Tunnel"</strong> — a secure connection that exposes a local service running on your device to the public internet via the fxtun infrastructure.</li>
          <li><strong>"Subdomain"</strong> — a unique hostname (e.g., <code>your-app.fxtun.dev</code>) assigned to an HTTP tunnel.</li>
          <li><strong>"Token"</strong> — an API credential used to authenticate tunnel connections.</li>
          <li><strong>"Plan"</strong> — a subscription tier defining the scope of Service available to you (Free, Base, or Pro).</li>
          <li><strong>"Traffic Inspector"</strong> — a built-in tool for monitoring HTTP requests and responses passing through your tunnels.</li>
        </ul>

        <h2>3. Description of the Service</h2>
        <p>
          fxtun provides secure reverse tunneling that allows you to expose local HTTP, TCP,
          and UDP services to the internet. The Service includes:
        </p>
        <ul>
          <li>HTTP tunneling with custom or random subdomains under <code>fxtun.dev</code>;</li>
          <li>TCP port forwarding with dynamically allocated public ports;</li>
          <li>UDP port forwarding;</li>
          <li>A web dashboard for managing tunnels, tokens, and reserved subdomains;</li>
          <li>Desktop GUI client for Windows, macOS, and Linux;</li>
          <li>Command-line client for all major platforms;</li>
          <li>Real-time traffic inspection for HTTP tunnels.</li>
        </ul>

        <h2>4. Account Registration</h2>
        <p>
          To use the Service, you must create an account. You agree to provide accurate and
          complete information during registration and to keep your account credentials secure.
        </p>
        <p>
          You are solely responsible for all activity that occurs under your account. You must
          notify us immediately at
          <a href="mailto:support@fxtun.ru">support@fxtun.ru</a> if you suspect
          unauthorized access to your account.
        </p>

        <h2>5. Free and Paid Plans</h2>
        <h3>5.1. Free Plan</h3>
        <p>
          The Free plan provides up to 3 concurrent tunnels with any available subdomain,
          no bandwidth limits, and no session timeout. The Free plan is available indefinitely
          and does not require a credit card.
        </p>
        <h3>5.2. Paid Plans</h3>
        <p>
          Paid plans (Base and Pro) offer additional capacity, reserved subdomains, custom domains,
          and priority features. Current pricing is available at
          <a href="https://fxtun.dev/#pricing">fxtun.dev/#pricing</a>.
        </p>
        <p>
          Subscriptions are billed monthly. Payment is processed through third-party payment
          providers. By subscribing, you authorize recurring charges to your selected payment
          method until you cancel.
        </p>

        <h2>6. Acceptable Use</h2>
        <p>You agree not to use the Service to:</p>
        <ul>
          <li>Violate any applicable law, regulation, or third-party rights;</li>
          <li>Distribute malware, phishing pages, or any form of malicious content;</li>
          <li>Conduct denial-of-service attacks or port scanning against third parties;</li>
          <li>Host or distribute illegal content, including but not limited to CSAM;</li>
          <li>Circumvent access controls or authentication of other systems;</li>
          <li>Relay spam or unsolicited bulk messages;</li>
          <li>Interfere with or disrupt the integrity of the Service or its infrastructure;</li>
          <li>Resell, redistribute, or sublicense access to the Service without prior written consent.</li>
        </ul>
        <p>
          We reserve the right to suspend or terminate your account immediately, without prior
          notice, if we reasonably determine that you have violated these terms.
        </p>

        <h2>7. Intellectual Property</h2>
        <p>
          The Service, including its software, design, logos, documentation, and all related
          intellectual property, is owned by Nocodo LTD or its licensors. These Terms do not
          grant you any right, title, or interest in the Service except for the limited right
          to use it as described herein.
        </p>
        <p>
          The open-source components of fxtun are licensed under their respective licenses
          as specified in the source code repositories.
        </p>

        <h2>8. Privacy and Data Processing</h2>
        <h3>8.1. Data We Collect</h3>
        <ul>
          <li><strong>Account data:</strong> email address, hashed password, account preferences;</li>
          <li><strong>Connection metadata:</strong> IP addresses, user agent strings, tunnel session timestamps;</li>
          <li><strong>Usage data:</strong> tunnel count, bandwidth statistics, feature usage;</li>
          <li><strong>Payment data:</strong> processed and stored exclusively by our payment providers — we do not store card details.</li>
        </ul>
        <h3>8.2. How We Use Your Data</h3>
        <ul>
          <li>To provide and maintain the Service;</li>
          <li>To authenticate your identity and manage your account;</li>
          <li>To process payments and manage subscriptions;</li>
          <li>To detect and prevent abuse and ensure platform security;</li>
          <li>To communicate essential service updates.</li>
        </ul>
        <h3>8.3. Data Retention</h3>
        <p>
          We retain your account data for the duration of your account plus 12 months after
          deletion. Connection metadata is retained for up to 90 days. You may request full
          data deletion by contacting <a href="mailto:support@fxtun.ru">support@fxtun.ru</a>.
        </p>
        <h3>8.4. Traffic Content</h3>
        <p>
          We do <strong>not</strong> inspect, log, or store the content of traffic passing through
          your tunnels. The Traffic Inspector feature operates exclusively in your browser session
          and on your device — tunnel payload data is not stored on our servers.
        </p>
        <h3>8.5. Third-Party Processors</h3>
        <p>We may share data with the following categories of third-party processors:</p>
        <ul>
          <li>Payment processors (for billing and subscription management);</li>
          <li>Infrastructure providers (for hosting and content delivery);</li>
          <li>Law enforcement (when required by applicable law or court order).</li>
        </ul>
        <h3>8.6. Cookies</h3>
        <p>
          The Service uses strictly necessary cookies for authentication and session management.
          We do not use tracking or advertising cookies.
        </p>

        <h2>9. Cancellation and Refunds</h2>
        <p>
          You may cancel your subscription at any time from your account dashboard. Upon
          cancellation, you retain access to paid features until the end of the current billing
          period.
        </p>
        <ul>
          <li><strong>Within 7 days of first payment:</strong> full refund, no questions asked.</li>
          <li><strong>After 7 days:</strong> pro-rata refund for unused full days remaining in the billing period.</li>
        </ul>
        <p>
          Refunds are issued to the original payment method within 14 business days.
          To request a refund, contact <a href="mailto:support@fxtun.ru">support@fxtun.ru</a>.
        </p>

        <h2>10. Service Availability and SLA</h2>
        <p>
          We make commercially reasonable efforts to maintain 99.9% uptime for the tunneling
          infrastructure. However, the Service is provided on an "as is" and "as available" basis.
        </p>
        <p>We are not liable for interruptions caused by:</p>
        <ul>
          <li>Scheduled maintenance (announced at least 24 hours in advance);</li>
          <li>Force majeure events, including natural disasters, wars, pandemics, or government actions;</li>
          <li>Third-party service failures (DNS providers, hosting providers, payment processors);</li>
          <li>Your network connectivity or local environment issues.</li>
        </ul>

        <h2>11. Limitation of Liability</h2>
        <p>
          To the maximum extent permitted by applicable law, Nocodo LTD shall not be liable
          for any indirect, incidental, special, consequential, or punitive damages, including
          but not limited to loss of profits, data, business opportunities, or goodwill,
          arising out of or related to your use of the Service.
        </p>
        <p>
          Our total aggregate liability for any claims arising from or related to the Service
          shall not exceed the amount you paid to us in the 12 months preceding the claim.
        </p>

        <h2>12. Indemnification</h2>
        <p>
          You agree to indemnify and hold harmless Nocodo LTD, its officers, directors, employees,
          and agents from any claims, damages, losses, liabilities, and expenses (including
          reasonable legal fees) arising from your use of the Service or violation of these Terms.
        </p>

        <h2>13. Modifications to the Terms</h2>
        <p>
          We may update these Terms from time to time. Material changes will be communicated
          via email or a prominent notice on the Service at least 30 days before they take effect.
          Continued use of the Service after the effective date constitutes acceptance of the
          updated Terms.
        </p>

        <h2>14. Termination</h2>
        <p>
          Either party may terminate this agreement at any time. You may do so by deleting
          your account or contacting support. We may terminate or suspend your access
          immediately for violation of these Terms, without prior notice or liability.
        </p>
        <p>
          Upon termination, your right to use the Service ceases immediately. Sections
          concerning intellectual property, limitation of liability, indemnification,
          and governing law survive termination.
        </p>

        <h2>15. Governing Law and Dispute Resolution</h2>
        <p>
          These Terms are governed by and construed in accordance with the laws of the
          Republic of Cyprus, without regard to its conflict of law provisions.
        </p>
        <p>
          Any disputes arising from or relating to these Terms or the Service shall first be
          attempted to be resolved through good-faith negotiation. If unresolved within 30 days,
          disputes shall be submitted to the exclusive jurisdiction of the courts of the
          Republic of Cyprus.
        </p>

        <h2>16. Severability</h2>
        <p>
          If any provision of these Terms is held to be invalid or unenforceable, the remaining
          provisions shall continue in full force and effect.
        </p>

        <h2>17. Contact Information</h2>
        <table class="w-full">
          <tbody>
            <tr>
              <td class="font-medium pr-4 py-1">Company:</td>
              <td>Nocodo LTD</td>
            </tr>
            <tr>
              <td class="font-medium pr-4 py-1">Jurisdiction:</td>
              <td>Republic of Cyprus</td>
            </tr>
            <tr>
              <td class="font-medium pr-4 py-1">Website:</td>
              <td><a href="https://nocodo.tech">nocodo.tech</a></td>
            </tr>
            <tr>
              <td class="font-medium pr-4 py-1">Email:</td>
              <td><a href="mailto:support@fxtun.ru">support@fxtun.ru</a></td>
            </tr>
            <tr>
              <td class="font-medium pr-4 py-1">Service:</td>
              <td><a href="https://fxtun.dev">fxtun.dev</a></td>
            </tr>
          </tbody>
        </table>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.prose h2 {
  @apply text-xl font-semibold mt-8 mb-4 text-foreground;
}

.prose h3 {
  @apply text-lg font-medium mt-6 mb-3 text-foreground;
}

.prose p {
  @apply mb-4 text-muted-foreground leading-relaxed;
}

.prose ul {
  @apply list-disc pl-6 mb-4 space-y-2 text-muted-foreground;
}

.prose a {
  @apply text-primary hover:underline;
}

.prose table {
  @apply mt-4;
}

.prose td {
  @apply py-2;
}

.prose code {
  @apply text-sm bg-surface px-1.5 py-0.5 rounded;
}
</style>
