CREATE TABLE profiles (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,

  display_name VARCHAR(120),
  date_of_birth DATE,

  gender VARCHAR(30),
  looking_for_gender VARCHAR(30),

  bio TEXT,
  height_cm INT,

  education VARCHAR(150),
  job_title VARCHAR(150),
  company VARCHAR(150),

  city VARCHAR(120),
  country VARCHAR(120),

  relationship_goal VARCHAR(50),

  show_age BOOLEAN NOT NULL DEFAULT TRUE,
  show_distance BOOLEAN NOT NULL DEFAULT TRUE,

  completion_percentage INT NOT NULL DEFAULT 0,
  profile_status VARCHAR(30) NOT NULL DEFAULT 'DRAFT',

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT profiles_gender_check CHECK (
    gender IS NULL OR gender IN ('MALE', 'FEMALE', 'NON_BINARY', 'OTHER')
  ),

  CONSTRAINT profiles_looking_for_gender_check CHECK (
    looking_for_gender IS NULL OR looking_for_gender IN ('MALE', 'FEMALE', 'EVERYONE')
  ),

  CONSTRAINT profiles_relationship_goal_check CHECK (
    relationship_goal IS NULL OR relationship_goal IN (
      'SERIOUS_RELATIONSHIP',
      'LONG_TERM',
      'FRIENDSHIP',
      'CASUAL',
      'NOT_SURE'
    )
  ),

  CONSTRAINT profiles_status_check CHECK (
    profile_status IN ('DRAFT', 'ACTIVE', 'PAUSED', 'UNDER_REVIEW', 'REJECTED')
  ),

  CONSTRAINT profiles_completion_check CHECK (
    completion_percentage >= 0 AND completion_percentage <= 100
  ),

  CONSTRAINT profiles_height_check CHECK (
    height_cm IS NULL OR (height_cm >= 100 AND height_cm <= 250)
  )
);

CREATE INDEX idx_profiles_user_id ON profiles(user_id);
CREATE INDEX idx_profiles_gender ON profiles(gender);
CREATE INDEX idx_profiles_looking_for_gender ON profiles(looking_for_gender);
CREATE INDEX idx_profiles_profile_status ON profiles(profile_status);

CREATE TRIGGER trg_profiles_set_updated_at
BEFORE UPDATE ON profiles
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE interest_categories (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  category_key VARCHAR(80) NOT NULL UNIQUE,
  name VARCHAR(100) NOT NULL UNIQUE,
  sort_order INT NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_interest_categories_is_active ON interest_categories(is_active);

CREATE TRIGGER trg_interest_categories_set_updated_at
BEFORE UPDATE ON interest_categories
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE interests (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  category_id BIGINT REFERENCES interest_categories(id) ON DELETE SET NULL,

  interest_key VARCHAR(80) NOT NULL UNIQUE,
  name VARCHAR(100) NOT NULL UNIQUE,
  icon VARCHAR(100),
  sort_order INT NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_interests_category_id ON interests(category_id);
CREATE INDEX idx_interests_is_active ON interests(is_active);

CREATE TRIGGER trg_interests_set_updated_at
BEFORE UPDATE ON interests
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE user_interests (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  interest_id BIGINT NOT NULL REFERENCES interests(id) ON DELETE CASCADE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT user_interests_unique UNIQUE (user_id, interest_id)
);

CREATE INDEX idx_user_interests_user_id ON user_interests(user_id);
CREATE INDEX idx_user_interests_interest_id ON user_interests(interest_id);

CREATE TABLE profile_prompt_questions (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  prompt_key VARCHAR(80) NOT NULL UNIQUE,
  question TEXT NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_profile_prompt_questions_active ON profile_prompt_questions(is_active);

CREATE TRIGGER trg_profile_prompt_questions_set_updated_at
BEFORE UPDATE ON profile_prompt_questions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE user_profile_prompts (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  prompt_question_id BIGINT NOT NULL REFERENCES profile_prompt_questions(id) ON DELETE CASCADE,

  answer TEXT NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT user_profile_prompts_unique UNIQUE (user_id, prompt_question_id)
);

CREATE INDEX idx_user_profile_prompts_user_id ON user_profile_prompts(user_id);

CREATE TRIGGER trg_user_profile_prompts_set_updated_at
BEFORE UPDATE ON user_profile_prompts
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE lifestyle_questions (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  question_key VARCHAR(80) NOT NULL UNIQUE,
  question TEXT NOT NULL,

  answer_type VARCHAR(30) NOT NULL DEFAULT 'SINGLE_CHOICE',
  options JSONB NOT NULL DEFAULT '[]'::jsonb,

  sort_order INT NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT lifestyle_questions_answer_type_check CHECK (
    answer_type IN ('SINGLE_CHOICE', 'MULTIPLE_CHOICE', 'TEXT')
  )
);

CREATE INDEX idx_lifestyle_questions_active ON lifestyle_questions(is_active);

CREATE TRIGGER trg_lifestyle_questions_set_updated_at
BEFORE UPDATE ON lifestyle_questions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE user_lifestyle_answers (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  lifestyle_question_id BIGINT NOT NULL REFERENCES lifestyle_questions(id) ON DELETE CASCADE,

  answer JSONB NOT NULL DEFAULT '{}'::jsonb,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT user_lifestyle_answers_unique UNIQUE (user_id, lifestyle_question_id)
);

CREATE INDEX idx_user_lifestyle_answers_user_id ON user_lifestyle_answers(user_id);

CREATE TRIGGER trg_user_lifestyle_answers_set_updated_at
BEFORE UPDATE ON user_lifestyle_answers
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
