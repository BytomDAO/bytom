.PHONY: clean

TARGET=tensority

LIBS=
UNAME := $(shell uname -s)
ifeq ($(UNAME),Darwin)
	LIBS += lib.dylib
	CC = /usr/local/opt/llvm/bin/clang++
else
	LIBS += lib.so
	CC = g++
endif

$(TARGET): cSimdTs.o
	rm -rf ../*.go
	cp *.go ../
	cd .. && go test -v && go test -bench=.
	rm *.o
	rm -rf ../*.go
	cp ../legacy/*.go ../

%.o: %.cpp
	$(CC) -o $@ -c $^ -std=c++11 -pthread -mavx2 -O3 -fopenmp -D_USE_OPENMP

clean:
	rm -f *.o *.so *.a