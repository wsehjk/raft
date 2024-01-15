#! /bin/bash 
GetTestCase() {
    case $1 in 
    "2A"|"2a") test='TestInitialElection2A TestReElection2A TestManyElections2A';;
    "2B"| "2b") test='TestBasicAgree2B TestRPCBytes2B TestFollowerFailure2B TestLeaderFailure2B  
                TestFailAgree2B  TestFailNoAgree2B TestConcurrentStarts2B TestRejoin2B TestBackup2B TestCount2B';;
    "2C"|"2c") test='TestPersist12C TestPersist22C TestPersist32C TestFigure82C TestUnreliableAgree2C TestFigure8Unreliable2C TestReliableChurn2C TestUnreliableChurn2C';;
    "2D"|"2d") test='TestSnapshotBasic2D TestSnapshotInstall2D TestSnapshotInstallUnreliable2D  TestSnapshotInstallCrash2D  TestSnapshotInstallUnCrash2D TestSnapshotAllCrash2D';;
    *) test=$1
    esac 
}
Usage() {
    echo "Usage: bash tesh.sh thread_num iteration test_case"        
}


if [[ $# -ne 3 ]]
then
    Usage
    exit
fi

GetTestCase $3
command="python3 dstest.py -r -p $1 -n $2 $test"
echo "$command"
$command