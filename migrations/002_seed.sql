-- Sample user: email test@example.com / password "password123"
-- (hash generated with bcrypt cost 10)
INSERT INTO users (email, name, password_hash) VALUES
('test@example.com', 'Test User', '$2b$10$5rcOSmrBNkc2KZjAtJVP1.3c04153bUn39Xma3L7i.0ldyLvnd.ge');

INSERT INTO problems (slug, title, description_md, difficulty, time_limit_ms, memory_limit_kb) VALUES
('two-sum', 'Two Sum',
 'Given an array of integers and a target, return the indices of the two numbers that add up to the target.',
 'easy', 2000, 128000);

INSERT INTO test_cases (problem_id, input, expected_output, is_sample, weight, ordinal)
SELECT id, '4
2 7 11 15
9', '0 1', true, 1, 0 FROM problems WHERE slug = 'two-sum';

INSERT INTO test_cases (problem_id, input, expected_output, is_sample, weight, ordinal)
SELECT id, '3
3 2 4
6', '1 2', false, 1, 1 FROM problems WHERE slug = 'two-sum';
