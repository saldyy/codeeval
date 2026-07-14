-- The status text column is no longer engine-specific (Judge0 -> Piston).
ALTER TABLE submission_results RENAME COLUMN judge0_status TO exec_status;
