.PHONY: clean

TARGET=cSimdTs

LIBS=
UNAME := $(shell uname -s)
ifeq ($(UNAME),Darwin)
	LIBS += lib.dylib
else
	LIBS += lib.so
endif

$(TARGET).dylib: cSimdTs.cpp
	rm -rf ../*.go
	cp *.go ../
	g++ -shared -o $@ $^ -std=c++11 -pthread -mavx2 -O3 -fPIC
	cp cSimdTs.dylib ../
	cd .. && go test
	rm -rf *.dylib
	rm -rf ../*.dylib
	rm -rf ../*.go
	cp ../legacy/*.go ../

clean:
	rm -f *.o *.so *.a *.dylib