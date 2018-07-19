TARGET = cSimdTs

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
    CXX = /usr/local/opt/llvm/bin/clang++
else
	CXX = g++
endif

CXXFLAGS = -std=c++11 -pthread -mavx2 -O3 -fopenmp -fPIC

.PHONY: clean

all: $(TARGET)

$(TARGET): $(TARGET).cpp
	$(CXX) -o $@.o -c $^ $(CXXFLAGS)

clean:
	rm -f *.o *.so *.a