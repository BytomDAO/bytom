#include <iostream>
#include <cstdio>
#include <map>
#include <mutex>
#include <signal.h>
#include "cSimdTs.h"
#include "BytomPoW.h"
#include "seed.h"

using namespace std;

BytomMatList16* matList_int16;
uint8_t result[32] = {0};
map <vector<uint8_t>, BytomMatList16*> seedCache;
static const int cacheSize = 42; //"Answer to the Ultimate Question of Life, the Universe, and Everything"
mutex mtx;

uint8_t *SimdTs(uint8_t blockheader[32], uint8_t seed[32]){
    mtx.lock();
    vector<uint8_t> seedVec(seed, seed + 32);

    if(seedCache.find(seedVec) != seedCache.end()) {
        // printf("\t---%s---\n", "Seed already exists in the cache.");
        matList_int16 = seedCache[seedVec];
    } else {
        uint32_t exted[32];
        extend(exted, seed); // extends seed to exted
        Words32 extSeed;
        init_seed(extSeed, exted);

        matList_int16 = new BytomMatList16;
        matList_int16->init(extSeed);

        seedCache.insert(make_pair(seedVec, matList_int16));
    }

    iter_mineBytom(blockheader, 32, result);

    if(seedCache.size() > cacheSize) {
        for(map<vector<uint8_t>, BytomMatList16*>::iterator it=seedCache.begin(); it!=seedCache.end(); ++it){
            delete it->second;
        }
        seedCache.clear();
    }

    mtx.unlock();
    return result;
}
