-- 2000ms was fine for the original stdin/stdout languages, but JVM cold
-- start + javac compilation under Piston routinely eats 1.5-2s on its own
-- before the user's code even runs, causing spurious timeouts for Java.
-- 3000ms is Piston's own configured ceiling for run_timeout - anything
-- higher gets rejected outright, not just ignored.
UPDATE problems SET time_limit_ms = 3000;

-- Same story for memory: the JVM baseline alone runs ~150-160MB, well over
-- the original 128000 KB (~125MB) cap, so Java submissions were getting
-- OOM-killed (exit 137) before user code even ran. 256000 KB leaves headroom.
UPDATE problems SET memory_limit_kb = 256000;
