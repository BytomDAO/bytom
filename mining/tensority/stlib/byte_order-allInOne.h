/* byte_order-allInOne.h */
#ifndef BYTE_ORDER_H
#define BYTE_ORDER_H
#include "ustd.h"
#include <stdlib.h>

#ifdef __GLIBC__
# include <endian.h>
#endif

#ifdef __cplusplus
extern "C" {
#endif

/* if x86 compatible cpu */
#if defined(i386) || defined(__i386__) || defined(__i486__) || \
	defined(__i586__) || defined(__i686__) || defined(__pentium__) || \
	defined(__pentiumpro__) || defined(__pentium4__) || \
	defined(__nocona__) || defined(prescott) || defined(__core2__) || \
	defined(__k6__) || defined(__k8__) || defined(__athlon__) || \
	defined(__amd64) || defined(__amd64__) || \
	defined(__x86_64) || defined(__x86_64__) || defined(_M_IX86) || \
	defined(_M_AMD64) || defined(_M_IA64) || defined(_M_X64)
/* detect if x86-64 instruction set is supported */
# if defined(_LP64) || defined(__LP64__) || defined(__x86_64) || \
	defined(__x86_64__) || defined(_M_AMD64) || defined(_M_X64)
#  define CPU_X64
# else
#  define CPU_IA32
# endif
#endif


/* detect CPU endianness */
#if (defined(__BYTE_ORDER) && defined(__LITTLE_ENDIAN) && \
		__BYTE_ORDER == __LITTLE_ENDIAN) || \
	(defined(__BYTE_ORDER__) && defined(__ORDER_LITTLE_ENDIAN__) && \
		__BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN__) || \
	defined(CPU_IA32) || defined(CPU_X64) || \
	defined(__ia64) || defined(__ia64__) || defined(__alpha__) || defined(_M_ALPHA) || \
	defined(vax) || defined(MIPSEL) || defined(_ARM_) || defined(__arm__)
# define CPU_LITTLE_ENDIAN
# define IS_BIG_ENDIAN 0
# define IS_LITTLE_ENDIAN 1
#elif (defined(__BYTE_ORDER) && defined(__BIG_ENDIAN) && \
		__BYTE_ORDER == __BIG_ENDIAN) || \
	(defined(__BYTE_ORDER__) && defined(__ORDER_BIG_ENDIAN__) && \
		__BYTE_ORDER__ == __ORDER_BIG_ENDIAN__) || \
	defined(__sparc) || defined(__sparc__) || defined(sparc) || \
	defined(_ARCH_PPC) || defined(_ARCH_PPC64) || defined(_POWER) || \
	defined(__POWERPC__) || defined(POWERPC) || defined(__powerpc) || \
	defined(__powerpc__) || defined(__powerpc64__) || defined(__ppc__) || \
	defined(__hpux)  || defined(_MIPSEB) || defined(mc68000) || \
	defined(__s390__) || defined(__s390x__) || defined(sel)
# define CPU_BIG_ENDIAN
# define IS_BIG_ENDIAN 1
# define IS_LITTLE_ENDIAN 0
#else
# error "Can't detect CPU architechture"
#endif

#ifndef __has_builtin
# define __has_builtin(x) 0
#endif

#define IS_ALIGNED_32(p) (0 == (3 & ((const char*)(p) - (const char*)0)))
#define IS_ALIGNED_64(p) (0 == (7 & ((const char*)(p) - (const char*)0)))

#if defined(_MSC_VER)
#define ALIGN_ATTR(n) __declspec(align(n))
#elif defined(__GNUC__)
#define ALIGN_ATTR(n) __attribute__((aligned (n)))
#else
#define ALIGN_ATTR(n) /* nothing */
#endif


#if defined(_MSC_VER) || defined(__BORLANDC__)
#define I64(x) x##ui64
#else
#define I64(x) x##ULL
#endif


#ifndef __STRICT_ANSI__
#define RHASH_INLINE inline
#elif defined(__GNUC__)
#define RHASH_INLINE __inline__
#else
#define RHASH_INLINE
#endif

/* define rhash_ctz - count traling zero bits */
#if (defined(__GNUC__) && __GNUC__ >= 4 || (__GNUC__ == 3 && __GNUC_MINOR__ >= 4)) || \
    (defined(__clang__) && __has_builtin(__builtin_ctz))
/* GCC >= 3.4 or clang */
# define rhash_ctz(x) __builtin_ctz(x)
#else
unsigned rhash_ctz(unsigned); /* define as function */
#endif

/* bswap definitions */
#if (defined(__GNUC__) && (__GNUC__ >= 4) && (__GNUC__ > 4 || __GNUC_MINOR__ >= 3)) || \
    (defined(__clang__) && __has_builtin(__builtin_bswap32) && __has_builtin(__builtin_bswap64))
/* GCC >= 4.3 or clang */
# define bswap_32(x) __builtin_bswap32(x)
# define bswap_64(x) __builtin_bswap64(x)
#elif (_MSC_VER > 1300) && (defined(CPU_IA32) || defined(CPU_X64)) /* MS VC */
# define bswap_32(x) _byteswap_ulong((unsigned long)x)
# define bswap_64(x) _byteswap_uint64((__int64)x)
#else
/* fallback to generic bswap definition */
static RHASH_INLINE uint32_t bswap_32(uint32_t x)
{
# if defined(__GNUC__) && defined(CPU_IA32) && !defined(__i386__) && !defined(RHASH_NO_ASM)
	__asm("bswap\t%0" : "=r" (x) : "0" (x)); /* gcc x86 version */
	return x;
# else
	x = ((x << 8) & 0xFF00FF00u) | ((x >> 8) & 0x00FF00FFu);
	return (x >> 16) | (x << 16);
# endif
}
static RHASH_INLINE uint64_t bswap_64(uint64_t x)
{
	union {
		uint64_t ll;
		uint32_t l[2];
	} w, r;
	w.ll = x;
	r.l[0] = bswap_32(w.l[1]);
	r.l[1] = bswap_32(w.l[0]);
	return r.ll;
}
#endif /* bswap definitions */

#ifdef CPU_BIG_ENDIAN
# define be2me_32(x) (x)
# define be2me_64(x) (x)
# define le2me_32(x) bswap_32(x)
# define le2me_64(x) bswap_64(x)

# define be32_copy(to, index, from, length) memcpy((to) + (index), (from), (length))
# define le32_copy(to, index, from, length) rhash_swap_copy_str_to_u32((to), (index), (from), (length))
# define be64_copy(to, index, from, length) memcpy((to) + (index), (from), (length))
# define le64_copy(to, index, from, length) rhash_swap_copy_str_to_u64((to), (index), (from), (length))
# define me64_to_be_str(to, from, length) memcpy((to), (from), (length))
# define me64_to_le_str(to, from, length) rhash_swap_copy_u64_to_str((to), (from), (length))

#else /* CPU_BIG_ENDIAN */
# define be2me_32(x) bswap_32(x)
# define be2me_64(x) bswap_64(x)
# define le2me_32(x) (x)
# define le2me_64(x) (x)

# define be32_copy(to, index, from, length) rhash_swap_copy_str_to_u32((to), (index), (from), (length))
# define le32_copy(to, index, from, length) memcpy((to) + (index), (from), (length))
# define be64_copy(to, index, from, length) rhash_swap_copy_str_to_u64((to), (index), (from), (length))
# define le64_copy(to, index, from, length) memcpy((to) + (index), (from), (length))
# define me64_to_be_str(to, from, length) rhash_swap_copy_u64_to_str((to), (from), (length))
# define me64_to_le_str(to, from, length) memcpy((to), (from), (length))
#endif /* CPU_BIG_ENDIAN */

/* ROTL/ROTR macros rotate a 32/64-bit word left/right by n bits */
#define ROTL32(dword, n) ((dword) << (n) ^ ((dword) >> (32 - (n))))
#define ROTR32(dword, n) ((dword) >> (n) ^ ((dword) << (32 - (n))))
#define ROTL64(qword, n) ((qword) << (n) ^ ((qword) >> (64 - (n))))
#define ROTR64(qword, n) ((qword) >> (n) ^ ((qword) << (64 - (n))))

#ifdef __cplusplus
} /* extern "C" */
#endif /* __cplusplus */

#endif /* BYTE_ORDER_H */


// Apdated from byte_order.c
/* byte_order.c - byte order related platform dependent routines,
 *
 * Copyright: 2008-2012 Aleksey Kravchenko <rhash.admin@gmail.com>
 *
 * Permission is hereby granted,  free of charge,  to any person  obtaining a
 * copy of this software and associated documentation files (the "Software"),
 * to deal in the Software without restriction,  including without limitation
 * the rights to  use, copy, modify,  merge, publish, distribute, sublicense,
 * and/or sell copies  of  the Software,  and to permit  persons  to whom the
 * Software is furnished to do so.
 *
 * This program  is  distributed  in  the  hope  that it will be useful,  but
 * WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
 * or FITNESS FOR A PARTICULAR PURPOSE.  Use this program  at  your own risk!
 */

#ifndef rhash_ctz

#  if _MSC_VER >= 1300 && (_M_IX86 || _M_AMD64 || _M_IA64) /* if MSVC++ >= 2002 on x86/x64 */
#  include <intrin.h>
#  pragma intrinsic(_BitScanForward)

/**
 * Returns index of the trailing bit of x.
 *
 * @param x the number to process
 * @return zero-based index of the trailing bit
 */
inline unsigned rhash_ctz(unsigned x)
{
	unsigned long index;
	unsigned char isNonzero = _BitScanForward(&index, x); /* MSVC intrinsic */
	return (isNonzero ? (unsigned)index : 0);
}
#  else /* _MSC_VER >= 1300... */

/**
 * Returns index of the trailing bit of a 32-bit number.
 * This is a plain C equivalent for GCC __builtin_ctz() bit scan.
 *
 * @param x the number to process
 * @return zero-based index of the trailing bit
 */
inline unsigned rhash_ctz(unsigned x)
{
	/* array for conversion to bit position */
	static unsigned char bit_pos[32] =  {
		0, 1, 28, 2, 29, 14, 24, 3, 30, 22, 20, 15, 25, 17, 4, 8,
		31, 27, 13, 23, 21, 19, 16, 7, 26, 12, 18, 6, 11, 5, 10, 9
	};

	/* The De Bruijn bit-scan was devised in 1997, according to Donald Knuth
	 * by Martin Lauter. The constant 0x077CB531UL is a De Bruijn sequence,
	 * which produces a unique pattern of bits into the high 5 bits for each
	 * possible bit position that it is multiplied against.
	 * See http://graphics.stanford.edu/~seander/bithacks.html
	 * and http://chessprogramming.wikispaces.com/BitScan */
	return (unsigned)bit_pos[((uint32_t)((x & -x) * 0x077CB531U)) >> 27];
}
#  endif /* _MSC_VER >= 1300... */
#endif /* rhash_ctz */

/**
 * Copy a memory block with simultaneous exchanging byte order.
 * The byte order is changed from little-endian 32-bit integers
 * to big-endian (or vice-versa).
 *
 * @param to the pointer where to copy memory block
 * @param index the index to start writing from
 * @param from  the source block to copy
 * @param length length of the memory block
 */
inline void rhash_swap_copy_str_to_u32(void* to, int index, const void* from, size_t length)
{
	/* if all pointers and length are 32-bits aligned */
	if ( 0 == (( (int)((char*)to - (char*)0) | ((char*)from - (char*)0) | index | length ) & 3) ) {
		/* copy memory as 32-bit words */
		const uint32_t* src = (const uint32_t*)from;
		const uint32_t* end = (const uint32_t*)((const char*)src + length);
		uint32_t* dst = (uint32_t*)((char*)to + index);
		for (; src < end; dst++, src++)
			*dst = bswap_32(*src);
	} else {
		const char* src = (const char*)from;
		for (length += index; (size_t)index < length; index++)
			((char*)to)[index ^ 3] = *(src++);
	}
}

/**
 * Copy a memory block with changed byte order.
 * The byte order is changed from little-endian 64-bit integers
 * to big-endian (or vice-versa).
 *
 * @param to     the pointer where to copy memory block
 * @param index  the index to start writing from
 * @param from   the source block to copy
 * @param length length of the memory block
 */
inline void rhash_swap_copy_str_to_u64(void* to, int index, const void* from, size_t length)
{
	/* if all pointers and length are 64-bits aligned */
	if ( 0 == (( (int)((char*)to - (char*)0) | ((char*)from - (char*)0) | index | length ) & 7) ) {
		/* copy aligned memory block as 64-bit integers */
		const uint64_t* src = (const uint64_t*)from;
		const uint64_t* end = (const uint64_t*)((const char*)src + length);
		uint64_t* dst = (uint64_t*)((char*)to + index);
		while (src < end) *(dst++) = bswap_64( *(src++) );
	} else {
		const char* src = (const char*)from;
		for (length += index; (size_t)index < length; index++) ((char*)to)[index ^ 7] = *(src++);
	}
}

/**
 * Copy data from a sequence of 64-bit words to a binary string of given length,
 * while changing byte order.
 *
 * @param to     the binary string to receive data
 * @param from   the source sequence of 64-bit words
 * @param length the size in bytes of the data being copied
 */
inline void rhash_swap_copy_u64_to_str(void* to, const void* from, size_t length)
{
	/* if all pointers and length are 64-bits aligned */
	if ( 0 == (( (int)((char*)to - (char*)0) | ((char*)from - (char*)0) | length ) & 7) ) {
		/* copy aligned memory block as 64-bit integers */
		const uint64_t* src = (const uint64_t*)from;
		const uint64_t* end = (const uint64_t*)((const char*)src + length);
		uint64_t* dst = (uint64_t*)to;
		while (src < end) *(dst++) = bswap_64( *(src++) );
	} else {
		size_t index;
		char* dst = (char*)to;
		for (index = 0; index < length; index++) *(dst++) = ((char*)from)[index ^ 7];
	}
}

/**
 * Exchange byte order in the given array of 32-bit integers.
 *
 * @param arr    the array to process
 * @param length array length
 */
inline void rhash_u32_mem_swap(unsigned *arr, int length)
{
	unsigned* end = arr + length;
	for (; arr < end; arr++) {
		*arr = bswap_32(*arr);
	}
}
