# Phase 15A PK Battle Contract

## Dependency Gate

Phase 15A PK Battle is blocked in this repo. This codebase does not own the required live, gift, wallet ledger, or live realtime domain modules.

Do not add these here:

```text
pk_battles migrations
internal/pk package
working PK APIs
fake live, gift, or wallet tables
PK scoring hooks
PK realtime events
admin PK monitoring routes
PK_ENABLED config
```

This repo exposes only disabled admin capabilities so clients can hide unavailable Games features:

```json
{
  "modules": {
    "games": false,
    "pkBattle": false,
    "greedyGame": false
  }
}
```

Permissions such as `games.read` are future-compatible authorization constants only. Permission does not imply backend capability.

## Required Backend Modules

Real PK Battle must be implemented only in the backend that owns:

```text
lives
live participants
gift transactions
gift receiver rows
wallet ledger
live realtime broadcaster
admin audit
admin game monitoring
```

The live/gift/wallet backend should enforce these MVP rules:

```text
one open PK per live
host cannot PK with themself
only live host can invite
only invited host can accept or reject
score comes only from committed gift receiver rows
gift retry cannot double-score
live end finalizes or cancels PK
admin can force-end, but cannot edit score or winner
```

## Future Data Model

The owning backend should add:

```text
pk_battles
pk_battle_contributions
pk_battle_events
```

`pk_battles` tracks the two live rooms, hosts, invite lifecycle, active timer, scores, winner, ended reason, and force-end admin. Open statuses are `INVITED` and `ACTIVE`; terminal statuses are `ENDED`, `REJECTED`, `CANCELLED`, `EXPIRED`, and `FORCE_ENDED`.

`pk_battle_contributions` links one gift receiver row to one PK score contribution. It must have a unique constraint on `gift_receiver_id` so duplicate gift retries cannot double-score.

`pk_battle_events` stores event history for invites, accept/reject/cancel, start, score update, normal end, expiry, and force-end.

## Future APIs

Public live routes, owned by the live backend:

```text
GET  /api/v1/lives/:liveId/pk/available-hosts
POST /api/v1/lives/:liveId/pk/invite
POST /api/v1/lives/:liveId/pk/:battleId/accept
POST /api/v1/lives/:liveId/pk/:battleId/reject
POST /api/v1/lives/:liveId/pk/:battleId/cancel
POST /api/v1/lives/:liveId/pk/:battleId/end
GET  /api/v1/lives/:liveId/pk/active
```

Admin routes, owned by the backend with game monitoring:

```text
GET  /api/v1/admin/pk-battles
GET  /api/v1/admin/pk-battles/:battleId
POST /api/v1/admin/pk-battles/:battleId/force-end
```

Do not register these routes in this repo while `pkBattle=false`.

## Future Realtime Events

The live realtime system should emit:

```text
pk:invite_received
pk:invite_cancelled
pk:invite_rejected
pk:started
pk:score_updated
pk:ended
pk:state
```

Clients must restore missed state with:

```text
GET /api/v1/lives/:liveId/pk/active
```

## Future Config and Rate Limits

The owning backend should define:

```env
PK_ENABLED=true
PK_DEFAULT_DURATION_SECONDS=300
PK_ALLOWED_DURATIONS=180,300,600
PK_INVITE_TTL_SECONDS=30
PK_MAX_INVITES_PER_10M=5
```

Recommended route limits:

```text
PK invite: 5 per 10 minutes per host
PK accept/reject/cancel: 20 per 10 minutes per host
PK active state: 60 per minute per user
Admin force-end: 20 per hour per admin
```

## Future Errors

The owning backend should expose:

```text
PK_NOT_FOUND
PK_NOT_ACTIVE
PK_ALREADY_ACTIVE_FOR_LIVE
PK_INVITE_NOT_FOUND
PK_INVITE_NOT_PENDING
PK_INVITE_EXPIRED
PK_SELF_INVITE_NOT_ALLOWED
PK_LIVE_NOT_ACTIVE
PK_OPPONENT_LIVE_NOT_ACTIVE
PK_HOST_REQUIRED
PK_OPPONENT_HOST_REQUIRED
PK_DURATION_INVALID
PK_ACCEPT_NOT_ALLOWED
PK_REJECT_NOT_ALLOWED
PK_CANCEL_NOT_ALLOWED
PK_END_NOT_ALLOWED
PK_FORCE_END_NOT_ALLOWED
PK_SCORE_RECORD_FAILED
```

## Future Tests

The real implementation should cover:

```text
duration validation
status transitions
winner and draw calculation
host invite/accept/reject/cancel/end permissions
one open PK per live
live end auto-finalization
gift scoring in the same transaction as gift commit
duplicate gift retry dedupe by gift receiver row
PK socket broadcasts
admin list/detail/force-end
admin audit on force-end
DB constraints and concurrent invite safety
```
