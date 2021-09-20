# dp-lock stress test

The below effectively acts as ONE instance of dp-dataset-api.

We create a number of go routines, each one taking the index int value of the for loop to identify the ‘the unique owner’

Each go routine does the following:

- (1) Set work_done_count = 0
- (2) Record start time for this go routine
- (3) Print out its unique number and time of getting lock
- (4) Pause 20ms
- (5) Release lock
- (6) Pause 10ms (we may reduce this to 5ms)
- (7) Work_done_count++
- (8) If work_done_count < 100 go back to step 3

If any go-routine times out, then the test will be aborted and all go-routines will exit.
