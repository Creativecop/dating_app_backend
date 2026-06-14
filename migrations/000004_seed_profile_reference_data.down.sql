UPDATE interests
SET is_active = FALSE
WHERE interest_key IN (
  'travel',
  'road_trips',
  'coffee',
  'cooking',
  'photography',
  'music',
  'gym',
  'football',
  'movies',
  'books',
  'fashion',
  'fitness'
);

UPDATE interest_categories
SET is_active = FALSE
WHERE category_key IN (
  'lifestyle',
  'creative',
  'sports',
  'food',
  'travel',
  'entertainment'
);

UPDATE profile_prompt_questions
SET is_active = FALSE
WHERE prompt_key IN (
  'perfect_weekend',
  'looking_for',
  'green_flag',
  'simple_pleasure',
  'life_goal',
  'best_travel_story'
);

UPDATE lifestyle_questions
SET is_active = FALSE
WHERE question_key IN (
  'smoking',
  'drinking',
  'workout',
  'pets',
  'communication_style'
);
