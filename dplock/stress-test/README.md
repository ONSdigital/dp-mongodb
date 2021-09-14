# dp-lock stress test


The below effectively acts as ONE instance of dp-dataset-api.

We create a number of go routines, each one taking the index int value of the for loop to identify the ‘the unique owner’

Each go routine does the following:

- (1) Set work_done_count = 0
- (2) Record start time for this go routine
- Get lock … if there is a timeout then panic as a first step, then refine.
- (3) Print out its unique number and time of getting lock
- (4) Pause 20ms
- (5) Release lock
- (6) Pause 10ms (we may reduce this to 5ms)
- (7) Work_done_count++
- (8) If work_done_count < 100 go back to step 3

TODO - figure out how to determine if no timeouts observed in any of the go routines to automate the test.

-=-=-

- Have just one caller.
- Have 2 callers.
- Have 10 go routines going against,
- Have 20 go routines going against,
- Have 50 go routines going against,
- Have 100 go routines going against,
- Have 1000 go routines going against,

With:
- A. a mock db ?
- B. local mongodb
- C. mongodb on dev

-=-=-

Observe and discuss results before doing anything else …

-=-=-

For example, we should experiment with different delays, etc

-=-=-

If the above looks OK, have an array (2 in size) of the lock structure for effectively 2 different instances running in parallel.

-=-=-

If the above looks OK, have an array (10 in size) of the lock structure for effectively 10 different instances running in parallel. (edited) 