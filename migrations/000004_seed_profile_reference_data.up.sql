INSERT INTO interest_categories (category_key, name, sort_order, is_active)
VALUES
  ('lifestyle', 'Lifestyle', 1, TRUE),
  ('creative', 'Creative', 2, TRUE),
  ('sports', 'Sports', 3, TRUE),
  ('food', 'Food & Drink', 4, TRUE),
  ('travel', 'Travel', 5, TRUE),
  ('entertainment', 'Entertainment', 6, TRUE)
ON CONFLICT (category_key) DO UPDATE SET
  name = EXCLUDED.name,
  sort_order = EXCLUDED.sort_order,
  is_active = EXCLUDED.is_active;

INSERT INTO interests (category_id, interest_key, name, icon, sort_order, is_active)
VALUES
  ((SELECT id FROM interest_categories WHERE category_key = 'travel'), 'travel', 'Travel', 'plane', 1, TRUE),
  ((SELECT id FROM interest_categories WHERE category_key = 'travel'), 'road_trips', 'Road Trips', 'car', 2, TRUE),
  ((SELECT id FROM interest_categories WHERE category_key = 'food'), 'coffee', 'Coffee', 'coffee', 3, TRUE),
  ((SELECT id FROM interest_categories WHERE category_key = 'food'), 'cooking', 'Cooking', 'chef-hat', 4, TRUE),
  ((SELECT id FROM interest_categories WHERE category_key = 'creative'), 'photography', 'Photography', 'camera', 5, TRUE),
  ((SELECT id FROM interest_categories WHERE category_key = 'creative'), 'music', 'Music', 'music', 6, TRUE),
  ((SELECT id FROM interest_categories WHERE category_key = 'sports'), 'gym', 'Gym', 'dumbbell', 7, TRUE),
  ((SELECT id FROM interest_categories WHERE category_key = 'sports'), 'football', 'Football', 'football', 8, TRUE),
  ((SELECT id FROM interest_categories WHERE category_key = 'entertainment'), 'movies', 'Movies', 'film', 9, TRUE),
  ((SELECT id FROM interest_categories WHERE category_key = 'entertainment'), 'books', 'Books', 'book-open', 10, TRUE),
  ((SELECT id FROM interest_categories WHERE category_key = 'lifestyle'), 'fashion', 'Fashion', 'shirt', 11, TRUE),
  ((SELECT id FROM interest_categories WHERE category_key = 'lifestyle'), 'fitness', 'Fitness', 'activity', 12, TRUE)
ON CONFLICT (interest_key) DO UPDATE SET
  category_id = EXCLUDED.category_id,
  name = EXCLUDED.name,
  icon = EXCLUDED.icon,
  sort_order = EXCLUDED.sort_order,
  is_active = EXCLUDED.is_active;

INSERT INTO profile_prompt_questions (prompt_key, question, sort_order, is_active)
VALUES
  ('perfect_weekend', 'A perfect weekend for me is...', 1, TRUE),
  ('looking_for', 'I am looking for...', 2, TRUE),
  ('green_flag', 'My biggest green flag is...', 3, TRUE),
  ('simple_pleasure', 'A simple pleasure I love is...', 4, TRUE),
  ('life_goal', 'One life goal I am working toward is...', 5, TRUE),
  ('best_travel_story', 'My best travel story is...', 6, TRUE)
ON CONFLICT (prompt_key) DO UPDATE SET
  question = EXCLUDED.question,
  sort_order = EXCLUDED.sort_order,
  is_active = EXCLUDED.is_active;

INSERT INTO lifestyle_questions (
  question_key,
  question,
  answer_type,
  options,
  sort_order,
  is_active
)
VALUES
  (
    'smoking',
    'Do you smoke?',
    'SINGLE_CHOICE',
    '["NO", "SOMETIMES", "YES", "PREFER_NOT_TO_SAY"]'::jsonb,
    1,
    TRUE
  ),
  (
    'drinking',
    'Do you drink?',
    'SINGLE_CHOICE',
    '["NO", "SOMETIMES", "SOCIALLY", "YES", "PREFER_NOT_TO_SAY"]'::jsonb,
    2,
    TRUE
  ),
  (
    'workout',
    'How often do you work out?',
    'SINGLE_CHOICE',
    '["NEVER", "SOMETIMES", "REGULARLY", "DAILY"]'::jsonb,
    3,
    TRUE
  ),
  (
    'pets',
    'Do you like pets?',
    'SINGLE_CHOICE',
    '["LOVE_PETS", "LIKE_PETS", "NOT_FOR_ME", "PREFER_NOT_TO_SAY"]'::jsonb,
    4,
    TRUE
  ),
  (
    'communication_style',
    'What is your communication style?',
    'SINGLE_CHOICE',
    '["TEXTING", "CALLS", "VOICE_NOTES", "IN_PERSON"]'::jsonb,
    5,
    TRUE
  )
ON CONFLICT (question_key) DO UPDATE SET
  question = EXCLUDED.question,
  answer_type = EXCLUDED.answer_type,
  options = EXCLUDED.options,
  sort_order = EXCLUDED.sort_order,
  is_active = EXCLUDED.is_active;
