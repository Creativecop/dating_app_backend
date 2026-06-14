INSERT INTO report_reasons (reason_code, title, description, applies_to, sort_order, is_active)
VALUES
  ('FAKE_PROFILE', 'Fake profile', 'The account appears to be fake or misleading.', ARRAY['USER', 'PROFILE'], 10, TRUE),
  ('HARASSMENT', 'Harassment', 'Abusive, bullying, or unwanted repeated behavior.', ARRAY['USER', 'PROFILE', 'MESSAGE', 'MATCH'], 20, TRUE),
  ('HATE_SPEECH', 'Hate speech', 'Content attacking protected characteristics.', ARRAY['USER', 'PROFILE', 'MESSAGE', 'MEDIA'], 30, TRUE),
  ('NUDITY_OR_SEXUAL_CONTENT', 'Nudity or sexual content', 'Explicit, sexual, or inappropriate content.', ARRAY['USER', 'PROFILE', 'MESSAGE', 'MEDIA'], 40, TRUE),
  ('SCAM_OR_SPAM', 'Scam or spam', 'Spam, fraud, phishing, or suspicious commercial behavior.', ARRAY['USER', 'PROFILE', 'MESSAGE'], 50, TRUE),
  ('UNDERAGE', 'Underage', 'The person appears to be underage.', ARRAY['USER', 'PROFILE', 'MEDIA'], 60, TRUE),
  ('VIOLENCE_OR_THREAT', 'Violence or threat', 'Threats, violence, coercion, or safety risk.', ARRAY['USER', 'PROFILE', 'MESSAGE'], 70, TRUE),
  ('IMPERSONATION', 'Impersonation', 'The account appears to impersonate someone else.', ARRAY['USER', 'PROFILE', 'MEDIA'], 80, TRUE),
  ('OTHER', 'Other', 'Another safety concern not listed here.', ARRAY['USER', 'PROFILE', 'MESSAGE', 'MEDIA', 'MATCH'], 90, TRUE)
ON CONFLICT (reason_code) DO UPDATE
SET
  title = EXCLUDED.title,
  description = EXCLUDED.description,
  applies_to = EXCLUDED.applies_to,
  sort_order = EXCLUDED.sort_order,
  is_active = TRUE,
  updated_at = NOW();
