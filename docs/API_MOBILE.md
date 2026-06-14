# Aura Mobile API Guide

Base URL for local development:

```text
http://localhost:8080
```

All JSON APIs use this response envelope:

```json
{
  "success": true,
  "statusCode": 200,
  "message": "OK",
  "data": {}
}
```

Errors use:

```json
{
  "success": false,
  "statusCode": 400,
  "message": "Validation failed",
  "error": {
    "code": "VALIDATION_ERROR",
    "details": {}
  }
}
```

Authenticated endpoints require:

```http
Authorization: Bearer <accessToken>
```

Do not store refresh tokens in plain shared preferences. Use platform secure storage.

## Auth

Request OTP:

```http
POST /api/v1/auth/request-otp
Content-Type: application/json
```

```json
{
  "channel": "WHATSAPP",
  "phone": "+8801XXXXXXXXX",
  "purpose": "LOGIN",
  "deviceId": "device-uuid"
}
```

Verify OTP:

```http
POST /api/v1/auth/verify-otp
```

```json
{
  "channel": "WHATSAPP",
  "phone": "+8801XXXXXXXXX",
  "purpose": "LOGIN",
  "code": "123456",
  "deviceId": "device-uuid",
  "deviceName": "iPhone 15",
  "platform": "ios",
  "fcmToken": "optional-fcm-token"
}
```

Refresh token:

```http
POST /api/v1/auth/refresh-token
```

```json
{
  "refreshToken": "<refreshToken>",
  "deviceId": "device-uuid"
}
```

Other auth endpoints:

```text
POST   /api/v1/auth/logout
POST   /api/v1/auth/logout-all
GET    /api/v1/auth/me
DELETE /api/v1/auth/account
```

## Profile Setup

Catalog endpoints:

```text
GET /api/v1/profile/catalog/interests
GET /api/v1/profile/catalog/prompts
GET /api/v1/profile/catalog/lifestyle-questions
```

Profile endpoints:

```text
GET   /api/v1/profile/me
PATCH /api/v1/profile/me
PUT   /api/v1/profile/interests
PUT   /api/v1/profile/prompts
PUT   /api/v1/profile/lifestyle
POST  /api/v1/profile/complete
```

Patch profile is partial. Send only changed fields:

```json
{
  "displayName": "Tanvir",
  "dateOfBirth": "1998-05-20",
  "gender": "MALE",
  "lookingForGender": "FEMALE",
  "bio": "I love travel and coffee.",
  "heightCm": 172,
  "education": "BSc in CSE",
  "jobTitle": "Flutter Developer",
  "company": "Neoscoder",
  "city": "Dhaka",
  "country": "Bangladesh",
  "relationshipGoal": "SERIOUS_RELATIONSHIP",
  "showAge": true,
  "showDistance": true
}
```

Update interests:

```json
{
  "interestIds": [1, 2, 3]
}
```

Update prompts:

```json
{
  "prompts": [
    {
      "promptQuestionId": 1,
      "answer": "A perfect weekend is a road trip with good music."
    }
  ]
}
```

## Media

Profile media endpoints:

```text
GET    /api/v1/profile/media/me
POST   /api/v1/profile/media/photos
POST   /api/v1/profile/media/intro-video
PATCH  /api/v1/profile/media/{mediaUuid}/primary
PATCH  /api/v1/profile/media/reorder
DELETE /api/v1/profile/media/{mediaUuid}
GET    /api/v1/media/{mediaUuid}/{variant}
```

Upload photos as multipart form data:

```text
file=<jpeg-or-png>
isPrimary=true|false
```

Allowed media variants:

```text
original
display
thumbnail
transcoded
```

Mobile clients should use `display` for profile detail and `thumbnail` for lists. Non-owners cannot access `original`.

## Location And Discovery Preferences

Update location:

```http
PUT /api/v1/location
```

```json
{
  "latitude": 23.8103,
  "longitude": 90.4125,
  "accuracyMeters": 25,
  "city": "Dhaka",
  "country": "Bangladesh",
  "source": "GPS"
}
```

Location endpoints:

```text
GET /api/v1/location/me
PUT /api/v1/location
```

Discovery preference endpoints:

```text
GET /api/v1/discovery/preferences
PUT /api/v1/discovery/preferences
GET /api/v1/discovery/readiness
```

Preferences are full replacement:

```json
{
  "minAge": 22,
  "maxAge": 35,
  "preferredGenders": ["FEMALE"],
  "maxDistanceKm": 30,
  "verifiedOnly": false,
  "showMeInDiscovery": true,
  "hideDistance": false
}
```

Readiness response helps drive setup UI:

```json
{
  "discoveryEligible": false,
  "missing": ["approvedPrimaryPhoto", "freshLocation"],
  "blocked": ["showMeInDiscoveryDisabled"]
}
```

## Discovery

Discovery endpoints:

```text
GET  /api/v1/discovery/feed?limit=20&cursor=<cursor>
GET  /api/v1/discovery/profiles/{userUuid}
POST /api/v1/discovery/impressions
POST /api/v1/discovery/actions
```

Action request:

```json
{
  "targetUserUuid": "target-user-uuid",
  "actionType": "LIKE",
  "clientActionId": "client-generated-uuid"
}
```

Allowed actions:

```text
LIKE
PASS
SUPER_LIKE
```

`clientActionId` is required for idempotency. Reusing the same key with a different target/action returns `IDEMPOTENCY_KEY_CONFLICT`.

Common discovery/action errors:

```text
DISCOVERY_NOT_READY
TARGET_NOT_DISCOVERABLE
ACTION_ALREADY_EXISTS
LIKE_LIMIT_REACHED
SUPER_LIKE_LIMIT_REACHED
```

## Matches And Chat

Match endpoints:

```text
GET  /api/v1/matches?limit=20&cursor=<cursor>
GET  /api/v1/matches/{matchUuid}
POST /api/v1/matches/{matchUuid}/seen
POST /api/v1/matches/{matchUuid}/unmatch
```

Chat endpoints:

```text
GET    /api/v1/chats?limit=20&cursor=<cursor>
GET    /api/v1/chats/{conversationUuid}/messages?limit=30&cursor=<cursor>
POST   /api/v1/chats/{conversationUuid}/messages
POST   /api/v1/chats/{conversationUuid}/read
DELETE /api/v1/messages/{messageUuid}
```

Send message:

```json
{
  "clientMessageId": "client-generated-uuid",
  "messageType": "TEXT",
  "body": "Hey, nice to meet you!"
}
```

Mark read:

```json
{
  "lastReadMessageUuid": "message-uuid"
}
```

## WebSocket

Connect:

```text
GET /ws
Authorization: Bearer <accessToken>
```

Mobile clients may also use:

```text
/ws?token=<accessToken>
```

Use access tokens only. Never send refresh tokens over WebSocket.

Client events:

```text
chat:message_delivered_ack
chat:typing_start
chat:typing_stop
chat:mark_read
```

Server events:

```text
chat:message_received
chat:message_delivered
chat:message_seen
chat:typing_started
chat:typing_stopped
chat:conversation_updated
chat:error
```

REST is the only supported message-send path. WebSocket `chat:send_message` is rejected with `UNSUPPORTED_EVENT`.

## Notifications

Device token:

```text
POST   /api/v1/devices/fcm-token
DELETE /api/v1/devices/fcm-token
```

Register FCM token:

```json
{
  "deviceId": "device-uuid",
  "fcmToken": "firebase-token",
  "deviceName": "Pixel 9",
  "platform": "android",
  "appVersion": "1.0.0",
  "osVersion": "Android 15"
}
```

Notification APIs:

```text
GET   /api/v1/notifications?limit=20&cursor=<cursor>
PATCH /api/v1/notifications/{notificationUuid}/read
PATCH /api/v1/notifications/read-all
GET   /api/v1/notification-settings
PUT   /api/v1/notification-settings
```

Push payloads are privacy-safe. They do not include message bodies, sender names, phone/email, or coordinates.

## Safety

Report and block endpoints:

```text
GET    /api/v1/reports/reasons
POST   /api/v1/reports
GET    /api/v1/reports/me
POST   /api/v1/safety/block-and-report
GET    /api/v1/safety/settings
PUT    /api/v1/safety/settings
POST   /api/v1/blocks/{userUuid}
DELETE /api/v1/blocks/{userUuid}
GET    /api/v1/blocks
```

Create report:

```json
{
  "targetType": "MESSAGE",
  "targetUuid": "target-uuid",
  "reasonCode": "HARASSMENT",
  "note": "Optional note",
  "blockUser": true
}
```

Supported target types:

```text
USER
PROFILE
MESSAGE
MEDIA
MATCH
```

## Subscriptions

Subscription endpoints:

```text
GET  /api/v1/subscriptions/plans
GET  /api/v1/subscriptions/me
GET  /api/v1/subscriptions/entitlements
GET  /api/v1/subscriptions/usage
GET  /api/v1/subscriptions/premium-status
POST /api/v1/subscriptions/manual-payment-requests
GET  /api/v1/subscriptions/manual-payment-requests
```

Create manual payment request:

```json
{
  "planCode": "AURA_PREMIUM_30",
  "paymentProvider": "BKASH",
  "paymentReference": "TRX123456",
  "payerPhone": "+8801XXXXXXXXX",
  "note": "Paid from personal bKash"
}
```

The app must not send amount. The backend snapshots plan price, currency, duration, and entitlements.

Entitlements response:

```json
{
  "isPremium": true,
  "planCode": "AURA_PREMIUM_30",
  "expiresAt": "2026-07-14T10:30:00Z",
  "entitlements": {
    "dailyLikeLimit": 300,
    "dailySuperLikeLimit": 10,
    "canUseAudioCall": true,
    "canUseVideoCall": true,
    "maxCallDurationSeconds": 1800,
    "dailyCallLimitSeconds": 7200,
    "canSeeWhoLikedMe": true,
    "canUseAdvancedFilters": true
  }
}
```

## Privacy Rules For Mobile

Never expect these in public/mobile profile responses:

```text
internal numeric IDs
phone
email
latitude
longitude
exact address
original media URLs for other users
admin audit data
moderation evidence snapshots
refresh tokens except auth responses
```

Use UUID fields as public identifiers.

## Common Client Handling

Recommended client behavior:

```text
401 UNAUTHORIZED: refresh token, then retry once.
403 DISCOVERY_NOT_READY: show setup/readiness screen.
409 IDEMPOTENCY_KEY_CONFLICT: treat as client bug; generate a new UUID for new action/message.
429 LIKE_LIMIT_REACHED: show upgrade/paywall or wait until next UTC day.
429 SUPER_LIKE_LIMIT_REACHED: show upgrade/paywall or wait until next UTC day.
```

Daily premium usage limits reset by UTC date.

