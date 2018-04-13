.PHONY: clean

TARGET=tensority

LIBS=
UNAME := $(shell uname -s)
ifeq ($(UNAME),Darwin)
	LIBS += lib.dylib
else
	LIBS += lib.so
endif

$(TARGET): cSimdTs.o
	cd .. && go test 

%.o: %.cpp
	g++ -o $@ -c $^ -std=c++11 -pthread -mavx2 -O3 -fopenmp -D_USE_OPENMP

clean:
	rm -f *.o *.so *.a