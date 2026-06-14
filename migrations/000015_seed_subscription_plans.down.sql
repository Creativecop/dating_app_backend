UPDATE subscription_plans
SET is_active = FALSE,
    updated_at = NOW()
WHERE plan_code IN ('AURA_PLUS_30', 'AURA_PREMIUM_30');
