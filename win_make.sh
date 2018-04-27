echo "Building bytomd to cmd/bytomd/bytomd for Win"
rm -f mining/tensority/*.go
cp mining/tensority/stlib/*.go mining/tensority/
g++ -o mining/tensority/stlib/cSimdTs.o -c mining/tensority/stlib/cSimdTs.cpp -std=c++11 -pthread -mavx2 -O3 -fopenmp -D_USE_OPENMP
go build -ldflags "-X github.com/bytom/version.GitCommit=`git rev-parse HEAD`" -o cmd/bytomd/bytomd.exe cmd/bytomd/main.go
rm -f mining/tensority/*.go
cp mining/tensority/legacy/*.go mining/tensority/
