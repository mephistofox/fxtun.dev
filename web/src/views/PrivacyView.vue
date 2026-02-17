<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useThemeStore, type ThemeMode } from '@/stores/theme'
import { setLocale, getLocale } from '@/i18n'
import { useSeo } from '@/composables/useSeo'

const themeStore = useThemeStore()
const { t, locale } = useI18n()

useSeo({ titleKey: 'seo.privacy.title', descriptionKey: 'seo.privacy.description' })

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

const lastUpdated = '17.02.2026'
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
        <h1 class="text-3xl font-bold mb-4">{{ t('legal.privacyTitle') }}</h1>
        <span class="text-sm text-muted-foreground">
          {{ t('legal.lastUpdated') }}: {{ lastUpdated }}
        </span>
      </div>

      <!-- Content -->
      <div class="prose prose-neutral dark:prose-invert max-w-none">
        <template v-if="locale === 'ru'">
          <h2>1. Введение</h2>
          <p>
            Настоящая Политика конфиденциальности описывает, как <strong>Nocodo LTD</strong>
            (далее «Компания», «мы», «нас» или «наш»), зарегистрированная в Республике Кипр,
            собирает, использует, раскрывает и защищает информацию пользователей сервиса fxTunnel
            (далее «Сервис»), включая веб-сайт <a href="https://fxtun.ru">fxtun.ru</a>,
            десктопные приложения, инструменты
            командной строки и все связанные API.
          </p>
          <p>
            Используя Сервис, вы соглашаетесь с условиями настоящей Политики конфиденциальности.
            Если вы не согласны, пожалуйста, не используйте Сервис.
          </p>

          <h2>2. Какую информацию мы собираем</h2>

          <h3>2.1. Данные аккаунта</h3>
          <p>При регистрации и использовании Сервиса мы собираем:</p>
          <ul>
            <li>Адрес электронной почты или номер телефона;</li>
            <li>Отображаемое имя (необязательно);</li>
            <li>Хэш пароля (мы не храним пароли в открытом виде);</li>
            <li>Настройки аккаунта и предпочтения.</li>
          </ul>

          <h3>2.2. Платёжные данные</h3>
          <p>
            Для обработки платежей мы используем сторонних платёжных провайдеров (Stripe, ЮKassa).
            Мы <strong>не</strong> храним номера банковских карт, CVV-коды или полные платёжные реквизиты.
            Мы сохраняем только:
          </p>
          <ul>
            <li>Идентификатор клиента в платёжной системе;</li>
            <li>Последние четыре цифры карты и статус срока действия;</li>
            <li>Историю платежей и статусы транзакций.</li>
          </ul>

          <h3>2.3. Метаданные подключений</h3>
          <p>При использовании туннелей мы собираем:</p>
          <ul>
            <li>IP-адреса клиентских подключений;</li>
            <li>Строки User Agent;</li>
            <li>Временные метки создания и завершения сессий туннелей;</li>
            <li>Тип туннеля (HTTP, TCP, UDP) и назначенные субдомены/порты.</li>
          </ul>

          <h3>2.4. Данные об использовании</h3>
          <ul>
            <li>Количество активных туннелей;</li>
            <li>Статистика трафика (объём, но не содержимое);</li>
            <li>Использование функций Сервиса.</li>
          </ul>

          <h3>2.5. Технические и журнальные данные</h3>
          <ul>
            <li>IP-адреса при входе в веб-панель;</li>
            <li>Информация о браузере и операционной системе;</li>
            <li>Журналы ошибок и производительности.</li>
          </ul>

          <h2>3. Как мы используем информацию</h2>
          <p>Собранные данные используются для:</p>
          <ul>
            <li>Предоставления и поддержки работы Сервиса;</li>
            <li>Аутентификации и управления аккаунтами;</li>
            <li>Обработки платежей и управления подписками;</li>
            <li>Выделения и маршрутизации субдоменов и портов;</li>
            <li>Обнаружения и предотвращения злоупотреблений;</li>
            <li>Отправки важных уведомлений о Сервисе (изменения тарифов, техобслуживание);</li>
            <li>Улучшения качества и производительности Сервиса.</li>
          </ul>

          <h2>4. Содержимое туннельного трафика</h2>
          <p>
            Мы <strong>не</strong> просматриваем, не записываем и не храним содержимое трафика,
            проходящего через ваши туннели. Данные передаются сквозным образом между вашим
            локальным сервисом и конечным пользователем.
          </p>
          <p>
            Инспектор трафика (Traffic Inspector) работает исключительно в вашем браузере
            и на вашем устройстве — данные туннельного трафика не передаются на наши серверы
            и не хранятся на них.
          </p>

          <h2>5. Передача данных третьим лицам</h2>
          <p>Мы можем передавать данные следующим категориям обработчиков:</p>
          <table class="w-full text-sm">
            <thead>
              <tr>
                <th class="text-left pr-4 py-2 font-medium">Провайдер</th>
                <th class="text-left pr-4 py-2 font-medium">Назначение</th>
                <th class="text-left py-2 font-medium">Передаваемые данные</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td class="pr-4 py-2">Stripe</td>
                <td class="pr-4 py-2">Обработка платежей (международных)</td>
                <td class="py-2">Имя, email, способ оплаты</td>
              </tr>
              <tr>
                <td class="pr-4 py-2">ЮKassa</td>
                <td class="pr-4 py-2">Обработка платежей (Россия)</td>
                <td class="py-2">Email/телефон, способ оплаты</td>
              </tr>
              <tr>
                <td class="pr-4 py-2">Провайдер инфраструктуры</td>
                <td class="pr-4 py-2">Хостинг серверов</td>
                <td class="py-2">Серверные конфигурации</td>
              </tr>
              <tr>
                <td class="pr-4 py-2">Провайдер email</td>
                <td class="pr-4 py-2">Уведомления</td>
                <td class="py-2">Email, статус аккаунта</td>
              </tr>
            </tbody>
          </table>
          <p>
            Мы <strong>не</strong> продаём, не сдаём в аренду и не передаём вашу персональную
            информацию третьим лицам в маркетинговых целях.
          </p>
          <p>
            Мы можем раскрыть информацию правоохранительным органам при наличии требования,
            основанного на применимом законодательстве или решении суда.
          </p>

          <h2>6. Сроки хранения данных</h2>
          <ul>
            <li><strong>Данные аккаунта:</strong> в течение срока действия аккаунта плюс 12 месяцев после удаления;</li>
            <li><strong>Метаданные подключений:</strong> до 90 дней, затем автоматически удаляются;</li>
            <li><strong>Платёжные записи:</strong> 7 лет (требование регулятора);</li>
            <li><strong>Серверные журналы:</strong> 90 дней, затем автоматически удаляются.</li>
          </ul>
          <p>
            Вы можете запросить полное удаление данных, обратившись по адресу
            <a href="mailto:support@nocodo.tech">support@nocodo.tech</a>.
          </p>

          <h2>7. Безопасность данных</h2>
          <p>Мы применяем следующие меры для защиты ваших данных:</p>
          <ul>
            <li>Шифрование TLS/HTTPS для всех передаваемых данных;</li>
            <li>Шифрование чувствительных данных при хранении (пароли, TOTP-секреты);</li>
            <li>Контроль доступа и принцип минимальных привилегий;</li>
            <li>Регулярные проверки безопасности и сканирование зависимостей;</li>
            <li>Двухфакторная аутентификация (TOTP) для защиты аккаунтов.</li>
          </ul>

          <h2>8. Ваши права</h2>
          <p>Вы имеете право:</p>
          <ul>
            <li><strong>Получить доступ</strong> к своим персональным данным;</li>
            <li><strong>Исправить</strong> неточные или неполные данные;</li>
            <li><strong>Удалить</strong> свои данные (с учётом законодательных обязательств по хранению);</li>
            <li><strong>Экспортировать</strong> свои данные в машиночитаемом формате;</li>
            <li><strong>Возразить</strong> против обработки данных.</li>
          </ul>
          <p>
            Для реализации своих прав обратитесь по адресу
            <a href="mailto:support@nocodo.tech">support@nocodo.tech</a>.
          </p>

          <h2>9. Положения для резидентов ЕС (GDPR)</h2>
          <h3>9.1. Правовые основания обработки</h3>
          <ul>
            <li><strong>Исполнение договора:</strong> обработка данных, необходимая для предоставления Сервиса;</li>
            <li><strong>Законный интерес:</strong> предотвращение злоупотреблений и улучшение Сервиса;</li>
            <li><strong>Юридическое обязательство:</strong> хранение платёжных записей в соответствии с требованиями законодательства.</li>
          </ul>
          <h3>9.2. Трансграничная передача данных</h3>
          <p>
            Данные туннелей обрабатываются на серверах в Европе. Платёжные данные, обрабатываемые
            Stripe, могут передаваться в США в рамках Соглашения об обработке данных и
            соответствующих гарантий.
          </p>

          <h2>10. Файлы cookie</h2>
          <p>
            Мы используем минимальное количество файлов cookie, строго необходимых для:
          </p>
          <ul>
            <li>Управления сессиями (аутентификация);</li>
            <li>Запоминания пользовательских настроек (тема, язык).</li>
          </ul>
          <p>
            Мы <strong>не</strong> используем сторонние трекинговые cookie, рекламные пиксели
            или аналитические трекеры.
          </p>

          <h2>11. Конфиденциальность несовершеннолетних</h2>
          <p>
            Сервис не предназначен для лиц младше 16 лет. Если мы узнаем, что собрали данные
            ребёнка, мы незамедлительно удалим их.
          </p>

          <h2>12. Изменения Политики</h2>
          <p>
            Мы можем обновлять настоящую Политику конфиденциальности. О существенных изменениях
            мы уведомим по электронной почте или заметным уведомлением в Сервисе не менее чем
            за 30 дней до вступления в силу. Продолжение использования Сервиса после даты
            вступления в силу означает принятие обновлённой Политики.
          </p>

          <h2>13. Контактная информация</h2>
          <table class="w-full">
            <tbody>
              <tr>
                <td class="font-medium pr-4 py-1">Компания:</td>
                <td>Nocodo LTD</td>
              </tr>
              <tr>
                <td class="font-medium pr-4 py-1">Юрисдикция:</td>
                <td>Республика Кипр</td>
              </tr>
              <tr>
                <td class="font-medium pr-4 py-1">Сайт:</td>
                <td><a href="https://nocodo.tech">nocodo.tech</a></td>
              </tr>
              <tr>
                <td class="font-medium pr-4 py-1">Email:</td>
                <td><a href="mailto:support@nocodo.tech">support@nocodo.tech</a></td>
              </tr>
              <tr>
                <td class="font-medium pr-4 py-1">Сервис:</td>
                <td><a href="https://fxtun.ru">fxtun.ru</a></td>
              </tr>
            </tbody>
          </table>
        </template>

        <template v-else>
          <h2>1. Introduction</h2>
          <p>
            This Privacy Policy describes how <strong>Nocodo LTD</strong>
            (hereinafter "Company", "we", "us", or "our"), a company registered in the
            Republic of Cyprus, collects, uses, discloses, and safeguards information of users
            of the fxTunnel service (the "Service"), including the website at
            <a href="https://fxtun.dev">fxtun.dev</a>,
            desktop applications, command-line tools, and all related APIs.
          </p>
          <p>
            By using the Service, you agree to the terms of this Privacy Policy.
            If you do not agree, please do not use the Service.
          </p>

          <h2>2. Information We Collect</h2>

          <h3>2.1. Account Information</h3>
          <p>When you register and use the Service, we collect:</p>
          <ul>
            <li>Email address or phone number;</li>
            <li>Display name (optional);</li>
            <li>Password hash (we do not store passwords in plain text);</li>
            <li>Account settings and preferences.</li>
          </ul>

          <h3>2.2. Payment Information</h3>
          <p>
            We use third-party payment processors (Stripe, YooKassa) to handle payments.
            We do <strong>not</strong> store your credit card numbers, CVVs, or full payment details.
            We only retain:
          </p>
          <ul>
            <li>Customer ID from the payment processor;</li>
            <li>Last four digits of your card and expiration status;</li>
            <li>Payment history and transaction statuses.</li>
          </ul>

          <h3>2.3. Connection Metadata</h3>
          <p>When you use tunnels, we collect:</p>
          <ul>
            <li>Client connection IP addresses;</li>
            <li>User agent strings;</li>
            <li>Tunnel session creation and termination timestamps;</li>
            <li>Tunnel type (HTTP, TCP, UDP) and assigned subdomains/ports.</li>
          </ul>

          <h3>2.4. Usage Data</h3>
          <ul>
            <li>Number of active tunnels;</li>
            <li>Traffic statistics (volume, not content);</li>
            <li>Service feature usage.</li>
          </ul>

          <h3>2.5. Technical and Log Data</h3>
          <ul>
            <li>IP addresses when accessing the web dashboard;</li>
            <li>Browser and operating system information;</li>
            <li>Error and performance logs.</li>
          </ul>

          <h2>3. How We Use Your Information</h2>
          <p>We use the collected data to:</p>
          <ul>
            <li>Provide and maintain the Service;</li>
            <li>Authenticate and manage user accounts;</li>
            <li>Process payments and manage subscriptions;</li>
            <li>Allocate and route subdomains and ports;</li>
            <li>Detect and prevent abuse;</li>
            <li>Send essential service notifications (pricing changes, maintenance);</li>
            <li>Improve service quality and performance.</li>
          </ul>

          <h2>4. Tunnel Traffic Content</h2>
          <p>
            We do <strong>not</strong> inspect, log, or store the content of traffic passing
            through your tunnels. Data is transmitted end-to-end between your local service
            and the end user.
          </p>
          <p>
            The Traffic Inspector feature operates exclusively in your browser session
            and on your device — tunnel payload data is not transmitted to or stored on
            our servers.
          </p>

          <h2>5. Data Sharing and Third Parties</h2>
          <p>We may share data with the following categories of processors:</p>
          <table class="w-full text-sm">
            <thead>
              <tr>
                <th class="text-left pr-4 py-2 font-medium">Provider</th>
                <th class="text-left pr-4 py-2 font-medium">Purpose</th>
                <th class="text-left py-2 font-medium">Data Shared</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td class="pr-4 py-2">Stripe</td>
                <td class="pr-4 py-2">Payment processing (international)</td>
                <td class="py-2">Name, email, payment method</td>
              </tr>
              <tr>
                <td class="pr-4 py-2">YooKassa</td>
                <td class="pr-4 py-2">Payment processing (Russia)</td>
                <td class="py-2">Email/phone, payment method</td>
              </tr>
              <tr>
                <td class="pr-4 py-2">Infrastructure provider</td>
                <td class="pr-4 py-2">Server hosting</td>
                <td class="py-2">Server configurations</td>
              </tr>
              <tr>
                <td class="pr-4 py-2">Email provider</td>
                <td class="pr-4 py-2">Notifications</td>
                <td class="py-2">Email, account status</td>
              </tr>
            </tbody>
          </table>
          <p>
            We do <strong>not</strong> sell, rent, or trade your personal information to
            third parties for marketing purposes.
          </p>
          <p>
            We may disclose information to law enforcement when required by applicable law
            or court order.
          </p>

          <h2>6. Data Retention</h2>
          <ul>
            <li><strong>Account data:</strong> duration of your account plus 12 months after deletion;</li>
            <li><strong>Connection metadata:</strong> up to 90 days, then automatically purged;</li>
            <li><strong>Billing records:</strong> 7 years (regulatory requirement);</li>
            <li><strong>Server logs:</strong> 90 days, then automatically purged.</li>
          </ul>
          <p>
            You may request full data deletion by contacting
            <a href="mailto:support@nocodo.tech">support@nocodo.tech</a>.
          </p>

          <h2>7. Data Security</h2>
          <p>We employ the following measures to protect your data:</p>
          <ul>
            <li>TLS/HTTPS encryption for all data in transit;</li>
            <li>Encryption at rest for sensitive data (passwords, TOTP secrets);</li>
            <li>Access controls and least-privilege principles;</li>
            <li>Regular security audits and dependency scanning;</li>
            <li>Two-factor authentication (TOTP) for account protection.</li>
          </ul>

          <h2>8. Your Rights</h2>
          <p>You have the right to:</p>
          <ul>
            <li><strong>Access</strong> your personal data;</li>
            <li><strong>Correct</strong> inaccurate or incomplete data;</li>
            <li><strong>Delete</strong> your data (subject to legal retention obligations);</li>
            <li><strong>Export</strong> your data in a machine-readable format;</li>
            <li><strong>Object</strong> to data processing.</li>
          </ul>
          <p>
            To exercise your rights, contact
            <a href="mailto:support@nocodo.tech">support@nocodo.tech</a>.
          </p>

          <h2>9. EU-Specific Provisions (GDPR)</h2>
          <h3>9.1. Legal Basis for Processing</h3>
          <ul>
            <li><strong>Contract performance:</strong> processing necessary to provide the Service;</li>
            <li><strong>Legitimate interest:</strong> abuse prevention and service improvement;</li>
            <li><strong>Legal obligation:</strong> retention of billing records as required by law.</li>
          </ul>
          <h3>9.2. Cross-Border Data Transfers</h3>
          <p>
            Tunnel data is processed on servers located in Europe. Payment data processed by
            Stripe may be transferred to the US under a Data Processing Agreement and
            appropriate safeguards.
          </p>

          <h2>10. Cookies</h2>
          <p>
            We use minimal cookies strictly necessary for:
          </p>
          <ul>
            <li>Session management (authentication);</li>
            <li>Remembering user preferences (theme, language).</li>
          </ul>
          <p>
            We do <strong>not</strong> use third-party tracking cookies, advertising pixels,
            or analytics trackers.
          </p>

          <h2>11. Children's Privacy</h2>
          <p>
            The Service is not intended for users under 16. If we learn that we have collected
            data from a child, we will delete it promptly.
          </p>

          <h2>12. Changes to This Policy</h2>
          <p>
            We may update this Privacy Policy from time to time. Material changes will be
            communicated via email or a prominent notice on the Service at least 30 days before
            they take effect. Continued use of the Service after the effective date constitutes
            acceptance of the updated Policy.
          </p>

          <h2>13. Contact Information</h2>
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
                <td><a href="mailto:support@nocodo.tech">support@nocodo.tech</a></td>
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

.prose th {
  @apply border-b border-border;
}

.prose code {
  @apply text-sm bg-surface px-1.5 py-0.5 rounded;
}
</style>
