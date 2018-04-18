#ifndef SCRYPT_H
#define SCRYPT_H

#include <stdint.h>
#include <assert.h>
#include <stdio.h>

struct Words16 {
  uint32_t w[16];
};

#define ROTL(a, b) (((a) << (b)) | ((a) >> (32 - (b))))

static inline void xor_salsa8(uint32_t B[16], const uint32_t Bx[16]) {
  uint32_t x00,x01,x02,x03,x04,x05,x06,x07,x08,x09,x10,x11,x12,x13,x14,x15;
  int i;

  x00 = (B[ 0] ^= Bx[ 0]);
  x01 = (B[ 1] ^= Bx[ 1]);
  x02 = (B[ 2] ^= Bx[ 2]);
  x03 = (B[ 3] ^= Bx[ 3]);
  x04 = (B[ 4] ^= Bx[ 4]);
  x05 = (B[ 5] ^= Bx[ 5]);
  x06 = (B[ 6] ^= Bx[ 6]);
  x07 = (B[ 7] ^= Bx[ 7]);
  x08 = (B[ 8] ^= Bx[ 8]);
  x09 = (B[ 9] ^= Bx[ 9]);
  x10 = (B[10] ^= Bx[10]);
  x11 = (B[11] ^= Bx[11]);
  x12 = (B[12] ^= Bx[12]);
  x13 = (B[13] ^= Bx[13]);
  x14 = (B[14] ^= Bx[14]);
  x15 = (B[15] ^= Bx[15]);
  for (i = 0; i < 8; i += 2) {
    /* Operate on columns. */
    x04 ^= ROTL(x00 + x12,  7);  x09 ^= ROTL(x05 + x01,  7);
    x14 ^= ROTL(x10 + x06,  7);  x03 ^= ROTL(x15 + x11,  7);

    x08 ^= ROTL(x04 + x00,  9);  x13 ^= ROTL(x09 + x05,  9);
    x02 ^= ROTL(x14 + x10,  9);  x07 ^= ROTL(x03 + x15,  9);

    x12 ^= ROTL(x08 + x04, 13);  x01 ^= ROTL(x13 + x09, 13);
    x06 ^= ROTL(x02 + x14, 13);  x11 ^= ROTL(x07 + x03, 13);

    x00 ^= ROTL(x12 + x08, 18);  x05 ^= ROTL(x01 + x13, 18);
    x10 ^= ROTL(x06 + x02, 18);  x15 ^= ROTL(x11 + x07, 18);

    /* Operate on rows. */
    x01 ^= ROTL(x00 + x03,  7);  x06 ^= ROTL(x05 + x04,  7);
    x11 ^= ROTL(x10 + x09,  7);  x12 ^= ROTL(x15 + x14,  7);

    x02 ^= ROTL(x01 + x00,  9);  x07 ^= ROTL(x06 + x05,  9);
    x08 ^= ROTL(x11 + x10,  9);  x13 ^= ROTL(x12 + x15,  9);

    x03 ^= ROTL(x02 + x01, 13);  x04 ^= ROTL(x07 + x06, 13);
    x09 ^= ROTL(x08 + x11, 13);  x14 ^= ROTL(x13 + x12, 13);

    x00 ^= ROTL(x03 + x02, 18);  x05 ^= ROTL(x04 + x07, 18);
    x10 ^= ROTL(x09 + x08, 18);  x15 ^= ROTL(x14 + x13, 18);
  }
  B[ 0] += x00;
  B[ 1] += x01;
  B[ 2] += x02;
  B[ 3] += x03;
  B[ 4] += x04;
  B[ 5] += x05;
  B[ 6] += x06;
  B[ 7] += x07;
  B[ 8] += x08;
  B[ 9] += x09;
  B[10] += x10;
  B[11] += x11;
  B[12] += x12;
  B[13] += x13;
  B[14] += x14;
  B[15] += x15;
}

struct Words32 {
  Words16 lo, hi;
  uint32_t get(uint32_t i) const {
    if(i<16) return lo.w[i];
    else if(i<32) return hi.w[i-16];
    else assert(false);
  }
  void xor_other(const Words32& other) {
    for(int i=0; i<16; i++) lo.w[i]^=other.lo.w[i];
    for(int i=0; i<16; i++) hi.w[i]^=other.hi.w[i];
  }
};

struct LTCMemory {
  Words32 w32[1024];
  const Words32& get(uint32_t i) const {
    assert(i<1024);
    return w32[i];
  }
  void printItems() {
    printf("\nprint scrypt items\n");
    for(int i = 0; i < 16; i++) {
      printf(" ");
      printf(" %u ", uint32_t(this->get(0).lo.w[i]));
    }
  }
  void scrypt(Words32& X) {
    for (int i = 0; i < 1024; i++) {
      w32[i]=X;
      xor_salsa8(X.lo.w, X.hi.w);
      xor_salsa8(X.hi.w, X.lo.w);
    }
    for (int i = 0; i < 1024; i++) {
      int j = X.hi.w[0] & 1023;
      X.xor_other(w32[j]);
      xor_salsa8(X.lo.w, X.hi.w);
      xor_salsa8(X.hi.w, X.lo.w);
    }
  }
};

#endif
