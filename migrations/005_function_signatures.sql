-- Switch grading from stdin/stdout string comparison to function-call
-- submissions: problems now declare a typed function signature, and test
-- cases carry structured call arguments + expected return value instead of
-- raw text.

ALTER TABLE problems ADD COLUMN function_signature JSONB;

UPDATE problems SET function_signature =
  '{"function_name": "twoSum", "params": [{"name": "nums", "type": "int[]"}, {"name": "target", "type": "int"}], "return_type": "int[]"}'
  WHERE slug = 'two-sum';
UPDATE problems SET function_signature =
  '{"function_name": "fizzBuzz", "params": [{"name": "n", "type": "int"}], "return_type": "string[]"}'
  WHERE slug = 'fizzbuzz';
UPDATE problems SET function_signature =
  '{"function_name": "reverseString", "params": [{"name": "s", "type": "string"}], "return_type": "string"}'
  WHERE slug = 'reverse-string';
UPDATE problems SET function_signature =
  '{"function_name": "isValid", "params": [{"name": "s", "type": "string"}], "return_type": "bool"}'
  WHERE slug = 'valid-parentheses';
UPDATE problems SET function_signature =
  '{"function_name": "maxSubArray", "params": [{"name": "nums", "type": "int[]"}], "return_type": "int"}'
  WHERE slug = 'max-subarray';

ALTER TABLE problems ALTER COLUMN function_signature SET NOT NULL;

ALTER TABLE test_cases ADD COLUMN args_json JSONB;
ALTER TABLE test_cases ADD COLUMN expected_return_json JSONB;

UPDATE test_cases tc SET args_json = v.args, expected_return_json = v.expected
FROM (VALUES
  ('two-sum', 0, '[[2,7,11,15],9]'::jsonb, '[0,1]'::jsonb),
  ('two-sum', 1, '[[3,2,4],6]'::jsonb, '[1,2]'::jsonb),
  ('fizzbuzz', 0, '[5]'::jsonb, '["1","2","Fizz","4","Buzz"]'::jsonb),
  ('fizzbuzz', 1, '[15]'::jsonb, '["1","2","Fizz","4","Buzz","Fizz","7","8","Fizz","Buzz","11","Fizz","13","14","FizzBuzz"]'::jsonb),
  ('reverse-string', 0, '["hello"]'::jsonb, '"olleh"'::jsonb),
  ('reverse-string', 1, '["algorithm"]'::jsonb, '"mhtirogla"'::jsonb),
  ('valid-parentheses', 0, '["()[]{}"]'::jsonb, 'true'::jsonb),
  ('valid-parentheses', 1, '["(]"]'::jsonb, 'false'::jsonb),
  ('max-subarray', 0, '[[-2,1,-3,4,-1,2,1,-5,4]]'::jsonb, '6'::jsonb),
  ('max-subarray', 1, '[[-1,-2,-3,-4,-5]]'::jsonb, '-1'::jsonb)
) AS v(slug, ordinal, args, expected)
JOIN problems p ON p.slug = v.slug
WHERE tc.problem_id = p.id AND tc.ordinal = v.ordinal;

ALTER TABLE test_cases ALTER COLUMN args_json SET NOT NULL;
ALTER TABLE test_cases ALTER COLUMN expected_return_json SET NOT NULL;
ALTER TABLE test_cases DROP COLUMN input;
ALTER TABLE test_cases DROP COLUMN expected_output;
