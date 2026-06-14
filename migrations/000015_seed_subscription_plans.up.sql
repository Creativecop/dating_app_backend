INSERT INTO subscription_plans (
  plan_code,
  name,
  description,
  price_amount,
  currency,
  duration_days,
  entitlements,
  sort_order,
  is_active
)
VALUES
(
  'AURA_PLUS_30',
  'Aura Plus Monthly',
  'More likes and basic premium benefits.',
  299,
  'BDT',
  30,
  '{
    "dailyLikeLimit": 100,
    "dailySuperLikeLimit": 3,
    "canUseAudioCall": false,
    "canUseVideoCall": false,
    "maxCallDurationSeconds": 0,
    "dailyCallLimitSeconds": 0,
    "canSeeWhoLikedMe": false,
    "canUseAdvancedFilters": false
  }'::jsonb,
  1,
  TRUE
),
(
  'AURA_PREMIUM_30',
  'Aura Premium Monthly',
  'Premium discovery and connection benefits.',
  599,
  'BDT',
  30,
  '{
    "dailyLikeLimit": 300,
    "dailySuperLikeLimit": 10,
    "canUseAudioCall": true,
    "canUseVideoCall": true,
    "maxCallDurationSeconds": 1800,
    "dailyCallLimitSeconds": 7200,
    "canSeeWhoLikedMe": true,
    "canUseAdvancedFilters": true
  }'::jsonb,
  2,
  TRUE
)
ON CONFLICT (plan_code) DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  price_amount = EXCLUDED.price_amount,
  currency = EXCLUDED.currency,
  duration_days = EXCLUDED.duration_days,
  entitlements = EXCLUDED.entitlements,
  sort_order = EXCLUDED.sort_order,
  is_active = EXCLUDED.is_active,
  updated_at = NOW();
