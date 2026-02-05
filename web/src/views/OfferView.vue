<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useThemeStore, type ThemeMode } from '@/stores/theme'
import { setLocale, getLocale } from '@/i18n'

const themeStore = useThemeStore()
const { t, locale } = useI18n()

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

const lastUpdated = '04.02.2026'
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
        <h1 class="text-3xl font-bold mb-4">{{ t('legal.offerTitle') }}</h1>
        <div class="flex items-center gap-4">
          <a
            href="/docs/offer.pdf"
            download
            class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z" clip-rule="evenodd" />
            </svg>
            {{ t('legal.downloadPdf') }}
          </a>
          <span class="text-sm text-muted-foreground">
            {{ t('legal.lastUpdated') }}: {{ lastUpdated }}
          </span>
        </div>
      </div>

      <!-- Content -->
      <div class="prose prose-neutral dark:prose-invert max-w-none">
        <template v-if="locale === 'ru'">
          <h2>1. Общие положения</h2>
          <p>
            В настоящей Публичной оферте содержатся условия заключения Договора об оказании услуг.
            Настоящей офертой признается предложение, адресованное одному или нескольким конкретным лицам,
            которое достаточно определенно и выражает намерение лица, сделавшего предложение, считать себя
            заключившим Договор с адресатом, которым будет принято предложение.
          </p>
          <p>
            Совершение указанных в настоящей Оферте действий является подтверждением согласия обеих Сторон
            заключить Договор об оказании услуг на условиях, в порядке и объеме, изложенных в настоящей Оферте.
          </p>
          <p>
            Нижеизложенный текст Публичной оферты является официальным публичным предложением Исполнителя,
            адресованный заинтересованному кругу лиц заключить Договор об оказании услуг в соответствии
            с положениями пункта 2 статьи 437 Гражданского кодекса РФ.
          </p>

          <h2>2. Термины и определения</h2>
          <ul>
            <li><strong>Договор</strong> — текст настоящей Оферты с Приложениями, акцептованный Заказчиком путем совершения конклюдентных действий.</li>
            <li><strong>Сайт Исполнителя</strong> — веб-сайт, доступный по адресу <a href="https://fxtun.ru">https://fxtun.ru</a></li>
            <li><strong>Услуга</strong> — услуга туннелирования, оказываемая Исполнителем Заказчику.</li>
            <li><strong>Стороны Договора</strong> — Исполнитель и Заказчик.</li>
          </ul>

          <h2>3. Предмет договора</h2>
          <p>
            Исполнитель обязуется оказать Заказчику услуги по предоставлению туннелей для доступа к локальным
            сервисам через интернет (HTTP, TCP, UDP туннели), а Заказчик обязуется оплатить их в размере,
            порядке и сроки, установленные настоящим Договором.
          </p>
          <p>
            Наименование, количество, порядок и иные условия оказания Услуг определяются на основании сведений,
            размещённых на сайте Исполнителя: <a href="https://fxtun.ru">https://fxtun.ru</a>
          </p>

          <h2>4. Стоимость услуг</h2>
          <p>
            Актуальные тарифы и стоимость услуг размещены на странице:
            <a href="https://fxtun.ru/#pricing">https://fxtun.ru/#pricing</a>
          </p>
          <p>
            Все расчеты по Договору производятся в безналичном порядке через платёжную систему ЮKassa.
            Подписка оплачивается ежемесячно путём автоматического списания (рекуррентные платежи).
          </p>

          <h2>5. Права и обязанности сторон</h2>
          <h3>5.1. Исполнитель обязуется:</h3>
          <ul>
            <li>Оказывать Услуги в соответствии с условиями настоящего Договора;</li>
            <li>Предоставлять Заказчику доступ к личному кабинету;</li>
            <li>Обеспечивать сохранение конфиденциальности персональных данных Заказчика.</li>
          </ul>
          <h3>5.2. Заказчик обязуется:</h3>
          <ul>
            <li>Предоставлять достоверную информацию о себе;</li>
            <li>Не использовать сервис для противоправной деятельности;</li>
            <li>Своевременно оплачивать услуги.</li>
          </ul>

          <h2>6. Условия возврата и отказа от услуг</h2>
          <ul>
            <li><strong>До начала оплаченного периода</strong> — полный возврат денежных средств.</li>
            <li><strong>После начала оплаченного периода</strong> — возврат пропорционально неиспользованному времени.</li>
            <li>Возврат осуществляется тем же способом, которым была произведена оплата, в течение <strong>14 рабочих дней</strong>.</li>
          </ul>
          <p>
            Для оформления возврата свяжитесь с нами:
            <a href="mailto:Mephistofox@ya.ru">Mephistofox@ya.ru</a> или
            <a href="https://t.me/mephistofx">@mephistofx</a> в Telegram.
          </p>

          <h2>7. Политика конфиденциальности</h2>
          <h3>7.1. Какие данные мы собираем:</h3>
          <ul>
            <li>Адрес электронной почты — для регистрации, авторизации и уведомлений;</li>
            <li>IP-адрес — для работы туннелей и обеспечения безопасности;</li>
            <li>Данные об использовании — статистика туннелей.</li>
          </ul>
          <h3>7.2. Цели обработки данных:</h3>
          <ul>
            <li>Предоставление услуг по созданию туннелей;</li>
            <li>Идентификация пользователя;</li>
            <li>Обеспечение безопасности сервиса;</li>
            <li>Выполнение требований законодательства РФ.</li>
          </ul>
          <h3>7.3. Хранение данных:</h3>
          <ul>
            <li>Данные хранятся на серверах в Российской Федерации;</li>
            <li>Срок хранения — в течение действия аккаунта + 1 год;</li>
            <li>Используется шифрование при передаче (TLS).</li>
          </ul>
          <h3>7.4. Передача данных третьим лицам:</h3>
          <p>Мы не передаём персональные данные третьим лицам, за исключением:</p>
          <ul>
            <li>Обработка платежей (ЮKassa);</li>
            <li>Требования законодательства РФ.</li>
          </ul>
          <h3>7.5. Права пользователя:</h3>
          <ul>
            <li>Запросить информацию о своих данных;</li>
            <li>Потребовать удаления данных;</li>
            <li>Отозвать согласие на обработку.</li>
          </ul>
          <h3>7.6. Cookies:</h3>
          <p>Сервис использует cookies для авторизации пользователей и сохранения настроек.</p>

          <h2>8. Форс-мажор</h2>
          <p>
            Стороны освобождаются от ответственности за неисполнение обязательств, если оно вызвано
            обстоятельствами непреодолимой силы: запретные действия властей, эпидемии, стихийные бедствия и т.д.
          </p>

          <h2>9. Ответственность сторон</h2>
          <p>
            В случае неисполнения обязательств по Договору, Стороны несут ответственность в соответствии
            с условиями настоящей Оферты и законодательством РФ.
          </p>

          <h2>10. Срок действия оферты</h2>
          <p>
            Оферта вступает в силу с момента размещения на Сайте Исполнителя и действует до момента её отзыва.
          </p>

          <h2>11. Дополнительные условия</h2>
          <ul>
            <li>Договор регулируется законодательством Российской Федерации;</li>
            <li>Досудебный порядок урегулирования споров обязателен;</li>
            <li>Язык договора — русский.</li>
          </ul>

          <h2>12. Реквизиты и контакты исполнителя</h2>
          <p>
            Полные реквизиты исполнителя указаны в
            <a href="/docs/offer.pdf" class="text-primary hover:underline">PDF-версии оферты</a>.
          </p>
          <table class="w-full">
            <tbody>
              <tr>
                <td class="font-medium pr-4 py-1">Email:</td>
                <td><a href="mailto:Mephistofox@ya.ru">Mephistofox@ya.ru</a></td>
              </tr>
              <tr>
                <td class="font-medium pr-4 py-1">Telegram:</td>
                <td><a href="https://t.me/mephistofx">@mephistofx</a></td>
              </tr>
              <tr>
                <td class="font-medium pr-4 py-1">Сайт:</td>
                <td><a href="https://fxtun.ru">https://fxtun.ru</a></td>
              </tr>
            </tbody>
          </table>
        </template>

        <template v-else>
          <h2>1. General Provisions</h2>
          <p>
            This Public Offer contains the terms of the Service Agreement. This offer is a proposal
            addressed to one or more specific persons, which is sufficiently definite and expresses
            the intention of the person making the proposal to consider themselves bound by a contract
            with the addressee who accepts the proposal.
          </p>

          <h2>2. Terms and Definitions</h2>
          <ul>
            <li><strong>Agreement</strong> — the text of this Offer with Appendices, accepted by the Customer.</li>
            <li><strong>Service Provider's Website</strong> — website available at <a href="https://fxtun.ru">https://fxtun.ru</a></li>
            <li><strong>Service</strong> — tunneling service provided by the Service Provider to the Customer.</li>
          </ul>

          <h2>3. Subject of the Agreement</h2>
          <p>
            The Service Provider undertakes to provide the Customer with tunneling services for accessing
            local services via the internet (HTTP, TCP, UDP tunnels), and the Customer undertakes to pay
            for them in the amount, manner and terms established by this Agreement.
          </p>

          <h2>4. Service Cost</h2>
          <p>
            Current rates and service costs are posted at:
            <a href="https://fxtun.ru/#pricing">https://fxtun.ru/#pricing</a>
          </p>

          <h2>5. Rights and Obligations</h2>
          <p>The Service Provider undertakes to provide services in accordance with this Agreement.</p>
          <p>The Customer undertakes to provide accurate information and pay for services on time.</p>

          <h2>6. Refund and Cancellation Policy</h2>
          <ul>
            <li><strong>Before the paid period starts</strong> — full refund.</li>
            <li><strong>After the paid period starts</strong> — refund proportional to unused time.</li>
            <li>Refunds are processed within <strong>14 business days</strong> using the same payment method.</li>
          </ul>
          <p>
            For refunds, contact us:
            <a href="mailto:Mephistofox@ya.ru">Mephistofox@ya.ru</a> or
            <a href="https://t.me/mephistofx">@mephistofx</a> on Telegram.
          </p>

          <h2>7. Privacy Policy</h2>
          <h3>7.1. Data We Collect:</h3>
          <ul>
            <li>Email address — for registration, authorization and notifications;</li>
            <li>IP address — for tunnel operation and security;</li>
            <li>Usage data — tunnel statistics.</li>
          </ul>
          <h3>7.2. Data Storage:</h3>
          <ul>
            <li>Data is stored on servers in the Russian Federation;</li>
            <li>Retention period — account duration + 1 year;</li>
            <li>TLS encryption is used for data transfer.</li>
          </ul>
          <h3>7.3. Third-Party Data Sharing:</h3>
          <p>We do not share personal data except for:</p>
          <ul>
            <li>Payment processing (YooKassa);</li>
            <li>Legal requirements of the Russian Federation.</li>
          </ul>

          <h2>8. Service Provider Details</h2>
          <p>
            Full details are available in the
            <a href="/docs/offer.pdf" class="text-primary hover:underline">PDF version</a>.
          </p>
          <table class="w-full">
            <tbody>
              <tr>
                <td class="font-medium pr-4 py-1">Email:</td>
                <td><a href="mailto:Mephistofox@ya.ru">Mephistofox@ya.ru</a></td>
              </tr>
              <tr>
                <td class="font-medium pr-4 py-1">Telegram:</td>
                <td><a href="https://t.me/mephistofx">@mephistofx</a></td>
              </tr>
              <tr>
                <td class="font-medium pr-4 py-1">Website:</td>
                <td><a href="https://fxtun.ru">https://fxtun.ru</a></td>
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
</style>
