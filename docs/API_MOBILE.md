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
GET /api/v1/profile/interests
GET /api/v1/profile/catalog/prompts
GET /api/v1/profile/prompts
GET /api/v1/profile/catalog/lifestyle-questions
GET /api/v1/profile/lifestyle
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

Media URLs are authenticated. Image requests to `/api/v1/media/{mediaUuid}/{variant}` must include the same header used by JSON APIs:

```http
Authorization: Bearer <accessToken>
```

Photo and intro video uploads return `202 Accepted` while processing continues. The upload response may only include `original` at first. Use the local picked file or the returned `original` URL for the owner's immediate preview, and switch to `display` or `thumbnail` after `GET /api/v1/profile/media/me` shows those variants.

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

## Admin Panel MVP

Admin APIs use separate admin JWTs. A normal mobile user token must never be used for `/api/v1/admin/*`.

Bootstrap the first admin from CLI only:

```bash
BOOTSTRAP_ADMIN_SECRET=strong_secret_min_32_chars \
go run ./cmd/bootstrap-admin --email=admin@example.com --name="Super Admin"
```

The bootstrap command prompts for a password unless `BOOTSTRAP_ADMIN_PASSWORD` is set. It works only while no active `SUPER_ADMIN` exists.

Admin auth endpoints:

```text
POST /api/v1/admin/auth/login
POST /api/v1/admin/auth/refresh-token
POST /api/v1/admin/auth/logout
GET  /api/v1/admin/auth/me
POST /api/v1/admin/auth/change-password
```

Login:

```json
{
  "email": "admin@example.com",
  "password": "admin-password"
}
```

Change password:

```json
{
  "currentPassword": "old-admin-password",
  "newPassword": "new-admin-password"
}
```

Changing an admin password revokes all admin sessions. The frontend should clear admin tokens and redirect to login.

Admins created by another admin start as `INVITED` with `mustChangePassword: true`. Invited admins can only call `POST /api/v1/admin/auth/change-password`; all other protected admin routes return `ADMIN_PASSWORD_CHANGE_REQUIRED` until the password is changed.

Admin role and lifecycle endpoints:

```text
GET    /api/v1/admin/roles
GET    /api/v1/admin/admin-users
GET    /api/v1/admin/admin-users/{adminUserUuid}
POST   /api/v1/admin/admin-users
POST   /api/v1/admin/admin-users/{adminUserUuid}/roles
DELETE /api/v1/admin/admin-users/{adminUserUuid}/roles/{role}
PATCH  /api/v1/admin/admin-users/{adminUserUuid}/status
```

Create admin:

```json
{
  "email": "ops@example.com",
  "name": "Ops Manager",
  "temporaryPassword": "TemporaryPassword@123",
  "roles": ["OPS_MANAGER"],
  "reason": "Ops team onboarding"
}
```

If `temporaryPassword` is omitted, the API generates one and returns it once in the create response.

Assign role:

```json
{
  "role": "OPS_MANAGER",
  "reason": "Coverage for live operations"
}
```

Remove role:

```json
{
  "reason": "Role no longer required"
}
```

Update admin status:

```json
{
  "status": "DISABLED",
  "reason": "Left company"
}
```

Admin statuses:

```text
INVITED  - can only change password
ACTIVE   - normal admin access
DISABLED - cannot log in
LOCKED   - cannot log in until status changes
```

Role assignments are the authorization source of truth. `SUPER_ADMIN` grants all permissions, but lower roles cannot assign/remove `SUPER_ADMIN`, modify a `SUPER_ADMIN`, or remove/disable the last active `SUPER_ADMIN`.

Admin capabilities:

```text
GET /api/v1/admin/capabilities
```

Capabilities are backend-derived from registered services and enabled modules; permissions do not imply that a module exists. In this repo, Trust & Safety report review is enabled, while games, PK Battle, Greedy Game, wallet, gift, agency, reseller, live, live-comment, and chat moderation modules are disabled until their domain packages exist.

Example response data:

```json
{
  "modules": {
    "trustSafety": true,
    "games": false,
    "pkBattle": false,
    "greedyGame": false,
    "wallet": false,
    "gift": false,
    "agency": false,
    "reseller": false,
    "live": false,
    "liveComments": false,
    "chatModeration": false
  }
}
```

Admin user operations:

```text
GET /api/v1/admin/users?search=017&status=ACTIVE&createdFrom=2026-01-01T00:00:00Z&createdTo=2026-01-31T23:59:59Z&limit=50&cursor=...
GET /api/v1/admin/users/{userId}
```

`userId` is the public user UUID. User detail includes available account/profile data, active restrictions, recent report summaries, and admin audit history for that user. Wallet and live summaries are `null` until those modules exist.

User restriction endpoints:

```text
GET    /api/v1/admin/users/{userId}/restrictions?status=ACTIVE
POST   /api/v1/admin/users/{userId}/restrictions
DELETE /api/v1/admin/users/{userId}/restrictions/{restrictionId}
```

Restriction list supports:

```text
status=ACTIVE
status=REVOKED
status=EXPIRED
```

Create restriction:

```json
{
  "restrictionType": "FULL_PLATFORM_BAN",
  "reason": "Repeated policy violations",
  "expiresAt": null
}
```

Revoke restriction:

```json
{
  "reason": "Appeal accepted"
}
```

Restriction behavior:

```text
FULL_PLATFORM_BAN blocks login, refresh, authenticated routes, socket connect, and direct chat.
COMMENT_BAN is reserved for future live comments and does not block direct chat.
Expired ACTIVE restrictions are ignored by enforcement and are moved to EXPIRED before replacement creation.
```

Admin report review endpoints:

```text
GET  /api/v1/admin/reports?status=PENDING&targetType=USER&severity=HIGH&limit=50&cursor=...
GET  /api/v1/admin/reports/{reportId}
POST /api/v1/admin/reports/{reportId}/review
```

`reportId` is the public report UUID. Report review requires `reports.review`, a pending report, and a reason.

Review without enforcement:

```json
{
  "decision": "REVIEWED",
  "reason": "Checked by trust and safety",
  "action": {
    "type": "NONE"
  }
}
```

Dismiss report:

```json
{
  "decision": "DISMISSED",
  "reason": "No violation found"
}
```

Restrict reported user:

```json
{
  "decision": "ACTIONED",
  "reason": "Abusive behavior confirmed",
  "action": {
    "type": "RESTRICT_USER",
    "restrictionType": "COMMENT_BAN",
    "expiresAt": "2026-07-17T00:00:00Z"
  }
}
```

`ACTIONED` requires a real executed backend action. In this repo, only `RESTRICT_USER` is available. `HIDE_COMMENT`, `FORCE_END_LIVE`, and `HIDE_CHAT_MESSAGE` return `REPORT_ACTION_NOT_ALLOWED`.

Live operation route contracts for future live module:

```text
GET  /api/v1/admin/lives/active
GET  /api/v1/admin/lives/{liveId}
POST /api/v1/admin/lives/{liveId}/force-end
```

These are not implemented until a real live domain service exists; admin handlers must call that service when added.

Future admin operation contracts for unavailable modules:

```text
Wallet:   audit, adjustment, transaction reversal
Gift:     catalog create/update/deactivate
Agency:   approve/reject/suspend/commission/settlement reports
Reseller: approve/reject/freeze/limits/commission/allocation/top-up reversal
```

These routes are not registered in this repo until real domain services and schemas exist.

Manual payment review endpoints:

```text
GET  /api/v1/admin/subscriptions/payment-requests?status=PENDING
GET  /api/v1/admin/subscriptions/payment-requests/{paymentRequestUuid}
POST /api/v1/admin/subscriptions/payment-requests/{paymentRequestUuid}/approve
POST /api/v1/admin/subscriptions/payment-requests/{paymentRequestUuid}/reject
```

Approve request:

```json
{
  "note": "Reference matched bKash statement"
}
```

Reject request:

```json
{
  "reason": "Payment reference was not found"
}
```

Payment review APIs require admin JWT plus payment review permissions. Approval activates or extends the user's subscription and writes audit data.

Audit log endpoints:

```text
GET /api/v1/admin/audit-logs
GET /api/v1/admin/audit-logs/{auditLogUuid}
```

Audit list filters:

```text
adminUserUuid
action
resourceType
resourceUuid
createdFrom
createdTo
limit
cursor
```

Audit log APIs require `admin.audit.read`.

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
