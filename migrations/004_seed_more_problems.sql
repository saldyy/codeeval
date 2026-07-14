-- A few more problems so /problems isn't just Two Sum.

INSERT INTO problems (slug, title, description_md, difficulty, time_limit_ms, memory_limit_kb) VALUES
('fizzbuzz', 'FizzBuzz',
 'Given an integer n, print the numbers from 1 to n, one per line. For multiples of 3 print "Fizz" instead of the number, for multiples of 5 print "Buzz", and for multiples of both print "FizzBuzz".',
 'easy', 2000, 128000),
('reverse-string', 'Reverse String',
 'Given a string, print it reversed.',
 'easy', 2000, 128000),
('valid-parentheses', 'Valid Parentheses',
 'Given a string containing only the characters ()[]{} , print "true" if the brackets are balanced and correctly nested, otherwise print "false".',
 'medium', 2000, 128000),
('max-subarray', 'Maximum Subarray',
 'Given an array of integers, print the largest possible sum of a contiguous subarray.',
 'medium', 2000, 128000);

-- FizzBuzz
INSERT INTO test_cases (problem_id, input, expected_output, is_sample, weight, ordinal)
SELECT id, '5', '1
2
Fizz
4
Buzz', true, 1, 0 FROM problems WHERE slug = 'fizzbuzz';

INSERT INTO test_cases (problem_id, input, expected_output, is_sample, weight, ordinal)
SELECT id, '15', '1
2
Fizz
4
Buzz
Fizz
7
8
Fizz
Buzz
11
Fizz
13
14
FizzBuzz', false, 1, 1 FROM problems WHERE slug = 'fizzbuzz';

-- Reverse String
INSERT INTO test_cases (problem_id, input, expected_output, is_sample, weight, ordinal)
SELECT id, 'hello', 'olleh', true, 1, 0 FROM problems WHERE slug = 'reverse-string';

INSERT INTO test_cases (problem_id, input, expected_output, is_sample, weight, ordinal)
SELECT id, 'algorithm', 'mhtirogla', false, 1, 1 FROM problems WHERE slug = 'reverse-string';

-- Valid Parentheses
INSERT INTO test_cases (problem_id, input, expected_output, is_sample, weight, ordinal)
SELECT id, '()[]{}', 'true', true, 1, 0 FROM problems WHERE slug = 'valid-parentheses';

INSERT INTO test_cases (problem_id, input, expected_output, is_sample, weight, ordinal)
SELECT id, '(]', 'false', false, 1, 1 FROM problems WHERE slug = 'valid-parentheses';

-- Maximum Subarray
INSERT INTO test_cases (problem_id, input, expected_output, is_sample, weight, ordinal)
SELECT id, '9
-2 1 -3 4 -1 2 1 -5 4', '6', true, 1, 0 FROM problems WHERE slug = 'max-subarray';

INSERT INTO test_cases (problem_id, input, expected_output, is_sample, weight, ordinal)
SELECT id, '5
-1 -2 -3 -4 -5', '-1', false, 1, 1 FROM problems WHERE slug = 'max-subarray';
