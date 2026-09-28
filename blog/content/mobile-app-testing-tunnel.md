---
title: "Тестирование мобильного приложения через туннель"
date: 2026-02-27T14:00:00+03:00
draft: false
description: "Реальный телефон не видит localhost компьютера. Показываем, как дать мобильному приложению HTTPS-адрес через fxTunnel и смотреть его запросы в Inspector."
tags: [mobile, testing, api, fxTunnel, ios, android, developer-tools]
image: ""
rubrics: [guides]
need: inspector
cover: ["localhost:8080", "fxtunnel http", "iPhone / Android"]
updated: 2026-09-27
---

## Проблема: реальное устройство не видит localhost

При разработке мобильного приложения API-сервер обычно работает на `localhost:8080`. Эмулятор iOS или Android может обращаться к нему напрямую, но настоящий телефон — нет. `localhost` указывает на само устройство, а не на ваш компьютер. Это значит, что для тестирования на реальном iPhone или Android вам нужен публичный URL, ведущий на ваш локальный сервер. Именно эту задачу решает [туннелирование](/blog/what-is-tunneling/).

Типичный сценарий: вы разрабатываете iOS-приложение с бэкендом на Go, Python или Node.js. В симуляторе всё работает, но на настоящем iPhone приложение получает ошибку `Could not connect to the server`. Причина проста — телефон не знает, что такое `localhost:8080` вашего Mac. Он обращается к своему собственному loopback-адресу и не находит там ничего.

Классический обходной путь — использовать IP-адрес компьютера в локальной сети (`192.168.1.42:8080`). Но у этого подхода три проблемы:

- **Нет HTTPS** — iOS App Transport Security блокирует HTTP-запросы, придётся добавлять исключения в `Info.plist`.
- **Привязка к сети** — работает только если телефон и компьютер в одной Wi-Fi сети. В кафе, дома или через мобильный интернет — не работает.
- **Ручная настройка** — IP-адрес меняется при смене сети, приходится обновлять конфигурацию вручную.

## Почему эмулятора недостаточно

Эмулятор — удобный инструмент для повседневной разработки, но он не заменяет тестирование на реальном устройстве. Критические различия проявляются в производительности, работе с оборудованием и поведении ОС — всё это невозможно воспроизвести в симуляторе.

- **Производительность** — эмулятор работает на мощном компьютере и не отражает реальные задержки, FPS и расход памяти на бюджетном Android-телефоне.
- **Жесты и тач** — мультитач, свайпы, 3D Touch / Haptic Touch, жесты навигации ведут себя иначе на реальном экране.
- **Камера и сенсоры** — сканирование QR-кодов, ARKit/ARCore, акселерометр, гироскоп, Face ID / Touch ID работают только на устройстве.
- **Push-уведомления** — APNs (Apple Push Notification service) не работает в симуляторе iOS. Для тестирования пушей нужен реальный iPhone.
- **Сеть** — реальное устройство работает через Wi-Fi или мобильный интернет с задержками и потерей пакетов. Эмулятор использует сеть компьютера напрямую.

Для полноценного тестирования нужны три вещи: локальный API, реальное устройство и способ связать их. Третий пункт -- как раз задача туннеля.

## Решение: fxTunnel даёт вашему API публичный HTTPS-адрес

fxTunnel создаёт [публичный URL для вашего localhost](/blog/expose-localhost-to-internet/) одной командой. Адрес работает по HTTPS с валидным сертификатом — это важно для iOS, где App Transport Security блокирует незащищённые соединения. Телефон обращается к `https://my-api.fxtun.ru`, запрос проходит через [сервер туннеля](/blog/fxtunnel-architecture/) и попадает на ваш `localhost:8080`.

### Быстрый старт

```bash
# Установка fxTunnel
curl -fsSL https://fxtun.ru/install.sh | bash

# Запуск API-сервера (пример на Go)
go run main.go
# → API listening on :8080

# Создание туннеля
fxtunnel http 8080
```

Вы получите публичный URL:

```console
Connecting to fxtunnel server...
Tunnel established!
HTTP:  http://my-api.fxtun.ru
HTTPS: https://my-api.fxtun.ru
Forwarding to localhost:8080
```

Без флага `--domain` при каждом новом запуске fxTunnel выдаёт случайный поддомен — адрес меняется, и в приложении пришлось бы каждый раз менять URL. Чтобы адрес не менялся, зарезервируйте поддомен и запускайте туннель с ним:

```bash
fxtunnel domains add my-api
fxtunnel http 8080 --domain my-api
```

Резервировать поддомены можно с тарифа Base — этого достаточно для одного мобильного API. На бесплатном тарифе `--domain my-api` сработает, пока это имя никто не занял и не зарезервировал, но это не гарантия. Теперь замените `http://localhost:8080` на `https://my-api.fxtun.ru` в конфигурации мобильного приложения — и реальное устройство получит доступ к вашему API.

## Настройка для iOS (Xcode + локальный API + туннель)

При разработке iOS-приложения в Xcode базовый URL API обычно хранится в конфигурации или файле `Info.plist`. Для переключения между localhost и туннелем достаточно вынести URL в переменную среды или конфигурационный файл.

### Конфигурация API-клиента в Swift

```swift
// APIConfig.swift
import Foundation

enum APIConfig {
    // Переключение между localhost (симулятор) и туннелем (реальное устройство)
    #if targetEnvironment(simulator)
    static let baseURL = "http://localhost:8080"
    #else
    static let baseURL = "https://my-api.fxtun.ru"
    #endif
}

// Использование
class APIClient {
    func fetchUsers() async throws -> [User] {
        let url = URL(string: "\(APIConfig.baseURL)/api/users")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode([User].self, from: data)
    }
}
```

### Почему HTTPS важен для iOS

App Transport Security (ATS) в iOS по умолчанию блокирует все HTTP-запросы без TLS. Разработчики часто добавляют исключения в `Info.plist`, чтобы обойти это ограничение при тестировании — но это плохая практика, которая может случайно попасть в продакшен-сборку.

fxTunnel автоматически выдаёт HTTPS-адрес с валидным TLS-сертификатом. Никаких исключений ATS добавлять не нужно — приложение работает с туннелем так же, как с продакшен-сервером. Тема HTTPS на localhost подробно раскрыта в [«HTTPS на localhost для разработки»](/blog/https-localhost-development/).

## Настройка для Android (Android Studio + локальный API + туннель)

В Android-разработке ситуация аналогичная: эмулятор может обращаться к `10.0.2.2` вместо `localhost`, но реальное устройство требует настоящий сетевой адрес.

### Конфигурация API-клиента в Kotlin

```kotlin
// ApiConfig.kt
object ApiConfig {
    // Для эмулятора: 10.0.2.2 — это хост-машина
    // Для реального устройства: URL туннеля
    val BASE_URL: String
        get() = if (BuildConfig.DEBUG && isEmulator()) {
            "http://10.0.2.2:8080"
        } else {
            "https://my-api.fxtun.ru"
        }

    private fun isEmulator(): Boolean {
        return (android.os.Build.FINGERPRINT.contains("generic")
                || android.os.Build.FINGERPRINT.contains("emulator"))
    }
}

// Retrofit setup
val retrofit = Retrofit.Builder()
    .baseUrl(ApiConfig.BASE_URL)
    .addConverterFactory(GsonConverterFactory.create())
    .build()
```

### Настройка Network Security Config

Начиная с Android 9 (API 28), незашифрованный HTTP-трафик заблокирован по умолчанию. При использовании fxTunnel этой проблемы нет — туннель предоставляет HTTPS. Но если вы используете локальный IP, потребуется `network_security_config.xml`:

```xml
<!-- res/xml/network_security_config.xml -->
<!-- НЕ нужно при использовании fxTunnel (HTTPS из коробки) -->
<network-security-config>
    <domain-config cleartextTrafficPermitted="true">
        <domain includeSubdomains="true">10.0.2.2</domain>
    </domain-config>
</network-security-config>
```

С fxTunnel этот файл не нужен — HTTPS работает автоматически.

## Настройка для React Native и Flutter

Кроссплатформенные фреймворки упрощают работу с API-адресом: достаточно изменить одну строку конфигурации. Но проблема доступа к localhost с реального устройства остаётся.

### React Native

```javascript
// config/api.js
import { Platform } from 'react-native';
import DeviceInfo from 'react-native-device-info';

const getBaseUrl = () => {
  if (__DEV__) {
    // Эмулятор: используем localhost / 10.0.2.2
    if (DeviceInfo.isEmulatorSync()) {
      return Platform.OS === 'android'
        ? 'http://10.0.2.2:8080'
        : 'http://localhost:8080';
    }
    // Реальное устройство: используем туннель
    return 'https://my-api.fxtun.ru';
  }
  // Продакшен
  return 'https://api.myapp.com';
};

export const API_BASE_URL = getBaseUrl();

// Использование с fetch
const response = await fetch(`${API_BASE_URL}/api/users`);
```

### Flutter (Dart)

```dart
// lib/config/api_config.dart
import 'dart:io';

class ApiConfig {
  static String get baseUrl {
    // В debug-режиме проверяем, запущено ли приложение на эмуляторе
    // Для реального устройства используем туннель
    const bool isRelease = bool.fromEnvironment('dart.vm.product');
    if (isRelease) {
      return 'https://api.myapp.com';
    }

    // Для реального устройства — URL туннеля
    // Для эмулятора — localhost
    // Управляется через --dart-define=API_URL=...
    return const String.fromEnvironment(
      'API_URL',
      defaultValue: 'https://my-api.fxtun.ru',
    );
  }
}

// Запуск с разными URL:
// flutter run --dart-define=API_URL=http://localhost:8080      (эмулятор)
// flutter run --dart-define=API_URL=https://my-api.fxtun.ru   (устройство)
```

```bash
# Запуск Flutter-приложения на реальном устройстве с URL туннеля
flutter run --dart-define=API_URL=https://my-api.fxtun.ru
```

## Тестирование push-уведомлений через туннель

Push-уведомления — один из главных аргументов в пользу реального устройства. APNs (Apple Push Notification service) и FCM (Firebase Cloud Messaging) не работают в симуляторах. Для полного цикла тестирования — отправка запроса на ваш API, формирование пуша на сервере, доставка на устройство — API должен быть доступен извне.

### Типичный flow

{{< flow caption="Запрос от приложения идёт через туннель до вашего API; саму доставку push берёт на себя APNs/FCM — уже не через туннель." >}}
- {label: мобильное приложение, title: POST /api/action}
- {link: HTTPS, kind: tls}
- {label: ваш API, title: клиент fxtunnel, sub: формирует push-уведомление, me: true}
- {link: APNs/FCM, kind: tls, note: минуя туннель}
- {label: устройство, title: push-уведомление}
{{< /flow >}}

Без туннеля для этого пришлось бы деплоить API на сервер после каждого изменения. С fxTunnel вы тестируете весь цикл локально:

```bash
# API работает на localhost:8080
# Туннель открыт
fxtunnel http 8080

# Тестируем отправку push-уведомления через API
curl -X POST https://my-api.fxtun.ru/api/send-push \
  -H "Content-Type: application/json" \
  -d '{"user_id": "123", "title": "Тест", "body": "Push через туннель"}'
```

Ваш локальный сервер получает запрос, формирует push через APNs/FCM, и через секунды уведомление приходит на реальный телефон — весь цикл без деплоя.

## Отладка: Inspector показывает запросы от мобильного приложения

Бывало такое: на экране телефона ошибка, а что именно приложение отправило -- непонятно? Charles Proxy и mitmproxy помогут, но потребуют настройки прокси на телефоне и установки корневых сертификатов. Встроенный Inspector в fxTunnel (доступен с тарифа Base, на бесплатном тарифе его нет) подходит к задаче иначе: он открывается на `127.0.0.1:4040` и показывает каждый запрос через туннель в веб-интерфейсе, обновляясь в реальном времени.

Что видно в Inspector:

- **URL и метод** — `POST /api/users`, `GET /api/feed`
- **Заголовки запроса** — `Authorization`, `Content-Type`, `User-Agent`
- **Тело запроса** — JSON-payload, который отправляет приложение
- **Статус ответа** — `200 OK`, `401 Unauthorized`, `500 Internal Server Error`
- **Тело ответа** — JSON, который вернул ваш API
- **Время ответа** — сколько миллисекунд занял запрос

### Replay для повторной отправки

Нашли баг? Не нужно воспроизводить сценарий на телефоне заново. Нажмите Replay в Inspector — запрос будет повторно отправлен на ваш API. Поставьте брейкпоинт в IDE и пошагово отладьте обработку. Это особенно полезно для сложных сценариев, где нужно пройти многошаговый flow в приложении.

Replay так же полезен при работе с вебхуками -- подробнее об этом в [«Тестирование вебхуков через туннель»](/blog/webhook-testing-with-tunnel/).

## Советы: certificate pinning и управление URL

### Certificate Pinning

Если ваше приложение использует SSL certificate pinning (закрепление сертификата), запросы через туннель будут заблокированы — сертификат fxTunnel не совпадёт с закреплённым. Решение:

```swift
// iOS: отключаем pinning для debug-сборок
#if DEBUG
// Используем стандартный URLSession без кастомного delegate
let session = URLSession.shared
#else
// Продакшен: включаем pinning
let session = URLSession(configuration: .default, delegate: PinningDelegate(), delegateQueue: nil)
#endif
```

```kotlin
// Android: отключаем pinning для debug-сборок
val client = if (BuildConfig.DEBUG) {
    OkHttpClient.Builder().build()  // без CertificatePinner
} else {
    OkHttpClient.Builder()
        .certificatePinner(
            CertificatePinner.Builder()
                .add("api.myapp.com", "sha256/AAAA...")
                .build()
        )
        .build()
}
```

### Управление API URL через конфигурацию

Не хардкодьте URL в приложении. Вынесите его в конфигурацию, чтобы переключаться между окружениями одной строкой:

```bash
# .env.development
API_URL=https://my-api.fxtun.ru

# .env.production
API_URL=https://api.myapp.com
```

API-сервер работает в Docker? Это тоже поддерживается -- настройка описана в [Docker + туннель](/blog/docker-tunnel-expose-containers/).

## Полный workflow: от кода до тестирования на телефоне

Вот пошаговый процесс разработки мобильного приложения с fxTunnel:

1. **Запустите API-сервер** на localhost.
2. **Откройте туннель** — `fxtunnel http 8080`.
3. **Укажите URL туннеля** в конфигурации мобильного приложения.
4. **Запустите приложение** на реальном устройстве через Xcode / Android Studio.
5. **Тестируйте** — все запросы идут через туннель на ваш localhost.
6. **Отлаживайте** — используйте Inspector для просмотра запросов и Replay для повторной отправки.
7. **Меняйте код** — изменения в API применяются мгновенно, без передеплоя.

```bash
# Терминал 1: API-сервер
cd ~/projects/my-api && go run main.go

# Терминал 2: туннель
fxtunnel http 8080

# Терминал 3: запуск на устройстве (Flutter)
cd ~/projects/my-app && flutter run --dart-define=API_URL=https://my-api.fxtun.ru
```

## Почему fxTunnel удобен для мобильной разработки

fxTunnel закрывает основные болевые точки мобильного тестирования: HTTPS из коробки (для iOS ATS и Android Network Security), закреплённый за поддоменом URL (не надо обновлять конфигурацию при каждом запуске) и Inspector для отладки запросов без прокси на телефоне.

| Задача | Без туннеля | С fxTunnel |
|---|---|---|
| Доступ к API с телефона | Только в одной Wi-Fi сети | Из любой сети |
| HTTPS для iOS ATS | Исключения в Info.plist | Автоматически |
| Просмотр запросов | Charles Proxy + сертификаты | Inspector в браузере (с тарифа Base) |
| Изменение кода API | Передеплой на сервер | Мгновенно, без деплоя |
| Push-уведомления | Деплой для каждого теста | Полный цикл локально |
| Смена IP при смене сети | Обновление конфигурации | Адрес закреплён за поддоменом (`--domain`) |

