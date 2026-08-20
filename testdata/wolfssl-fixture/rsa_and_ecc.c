// testdata/wolfssl-fixture/rsa_and_ecc.c
// Fixture for testing wolfSSL cryptographic pattern detection
// Does not require wolfSSL headers to be installed for static analysis

typedef struct { unsigned char data[256]; } RsaKey;
typedef struct { unsigned char data[256]; } ecc_key;
typedef struct { unsigned char data[256]; } WC_RNG;
typedef unsigned char byte;
typedef unsigned int word32;

// Forward declarations of wolfSSL functions
int wc_MakeRsaKey(RsaKey* key, int size, long e, WC_RNG* rng);
int wc_ecc_make_key(WC_RNG* rng, int keysize, ecc_key* key);
int wc_RsaSSL_Sign(byte* in, word32 inLen, byte* out, word32 outLen, RsaKey* key, WC_RNG* rng);
int wc_ecc_sign_hash(byte* hash, word32 hashLen, byte* sig, word32* sigLen, WC_RNG* rng, ecc_key* key);

void generate_and_sign(void) {
    RsaKey rsaKey;
    ecc_key eccKey;
    WC_RNG rng;
    byte msg[64] = {0};
    word32 msgLen = sizeof(msg);
    byte hash[32] = {0};
    word32 hashLen = sizeof(hash);
    byte sig[256];
    word32 sigLen = sizeof(sig);

    wc_MakeRsaKey(&rsaKey, 2048, 65537, &rng);
    wc_ecc_make_key(&rng, 32, &eccKey);

    wc_RsaSSL_Sign(msg, msgLen, sig, &sigLen, &rsaKey, &rng);
    wc_ecc_sign_hash(hash, hashLen, sig, &sigLen, &rng, &eccKey);
}
