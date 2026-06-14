DROP TRIGGER IF EXISTS trg_user_lifestyle_answers_set_updated_at ON user_lifestyle_answers;
DROP TABLE IF EXISTS user_lifestyle_answers;

DROP TRIGGER IF EXISTS trg_lifestyle_questions_set_updated_at ON lifestyle_questions;
DROP TABLE IF EXISTS lifestyle_questions;

DROP TRIGGER IF EXISTS trg_user_profile_prompts_set_updated_at ON user_profile_prompts;
DROP TABLE IF EXISTS user_profile_prompts;

DROP TRIGGER IF EXISTS trg_profile_prompt_questions_set_updated_at ON profile_prompt_questions;
DROP TABLE IF EXISTS profile_prompt_questions;

DROP TABLE IF EXISTS user_interests;

DROP TRIGGER IF EXISTS trg_interests_set_updated_at ON interests;
DROP TABLE IF EXISTS interests;

DROP TRIGGER IF EXISTS trg_interest_categories_set_updated_at ON interest_categories;
DROP TABLE IF EXISTS interest_categories;

DROP TRIGGER IF EXISTS trg_profiles_set_updated_at ON profiles;
DROP TABLE IF EXISTS profiles;
