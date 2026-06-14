UPDATE report_reasons
SET is_active = FALSE,
    updated_at = NOW()
WHERE reason_code IN (
  'FAKE_PROFILE',
  'HARASSMENT',
  'HATE_SPEECH',
  'NUDITY_OR_SEXUAL_CONTENT',
  'SCAM_OR_SPAM',
  'UNDERAGE',
  'VIOLENCE_OR_THREAT',
  'IMPERSONATION',
  'OTHER'
);
