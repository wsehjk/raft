python3 dstest.py -p 8 -n 100 TestInitialElection2A TestReElection2A TestManyElections2A 
python3 dstest.py -r -p 8 -n 100 TestBasicAgree2B TestRPCBytes2B TestFollowerFailure2B TestLeaderFailure2B \
        TestFailAgree2B  TestFailNoAgree2B TestConcurrentStarts2B TestRejoin2B TestBackup2B TestCount2B 