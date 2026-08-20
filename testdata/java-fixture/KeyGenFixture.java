import java.security.KeyPairGenerator;
import java.security.KeyPair;
import java.security.Signature;
import java.security.PrivateKey;
import java.security.PublicKey;
import javax.crypto.KeyAgreement;

/**
 * Comprehensive test fixture for Java quantum-vulnerability detection.
 * Covers RSA/ECC key generation, signing, and key agreement patterns.
 */
public class KeyGenFixture {

    // ========== RSA KEY GENERATION ==========

    // Test case 1: Imported RSA keygen with literal "RSA"
    public KeyPair generateRsaKeyImported() throws Exception {
        KeyPairGenerator kpg = KeyPairGenerator.getInstance("RSA");
        kpg.initialize(2048);
        return kpg.generateKeyPair();
    }

    // Test case 2: Fully qualified RSA keygen with literal "RSA"
    public KeyPair generateRsaKeyFullyQualified() throws Exception {
        KeyPairGenerator kpg = java.security.KeyPairGenerator.getInstance("RSA");
        kpg.initialize(2048);
        return kpg.generateKeyPair();
    }

    // Test case 3: RSA keygen with provider parameter
    public KeyPair generateRsaKeyWithProvider() throws Exception {
        KeyPairGenerator kpg = KeyPairGenerator.getInstance("RSA", "BC");
        kpg.initialize(2048);
        return kpg.generateKeyPair();
    }

    // Test case 4: RSA keygen via constant propagation
    private static final String RSA_ALGO = "RSA";
    
    public KeyPair generateRsaKeyViaConstant() throws Exception {
        KeyPairGenerator kpg = KeyPairGenerator.getInstance(RSA_ALGO);
        kpg.initialize(2048);
        return kpg.generateKeyPair();
    }

    // Test case 5: Unresolved algorithm — configuration-driven (needs manual review)
    private String algorithmFromConfig;

    public KeyPair generateKeyWithConfigAlgorithm() throws Exception {
        KeyPairGenerator kpg = KeyPairGenerator.getInstance(algorithmFromConfig);
        kpg.initialize(2048);
        return kpg.generateKeyPair();
    }

    // Test case 6: Unresolved algorithm — method parameter (needs manual review)
    public KeyPair generateKeyWithDynamicAlgorithm(String algorithm) throws Exception {
        KeyPairGenerator kpg = KeyPairGenerator.getInstance(algorithm);
        kpg.initialize(2048);
        return kpg.generateKeyPair();
    }

    // ========== ECC KEY GENERATION ==========

    // Test case 7: Imported ECC keygen with literal "EC"
    public KeyPair generateEcdsaKeyImported() throws Exception {
        KeyPairGenerator kpg = KeyPairGenerator.getInstance("EC");
        kpg.initialize(256);
        return kpg.generateKeyPair();
    }

    // Test case 8: Fully qualified ECC keygen
    public KeyPair generateEcdsaKeyFullyQualified() throws Exception {
        KeyPairGenerator kpg = java.security.KeyPairGenerator.getInstance("EC");
        kpg.initialize(256);
        return kpg.generateKeyPair();
    }

    // ========== RSA SIGNING ==========

    // Test case 9: RSA signing with imported Signature and SHA256withRSA
    public byte[] signWithRsaSha256(byte[] data, PrivateKey privKey) throws Exception {
        Signature sig = Signature.getInstance("SHA256withRSA");
        sig.initSign(privKey);
        sig.update(data);
        return sig.sign();
    }

    // Test case 10: RSA signing with fully qualified form
    public byte[] signWithRsaSha256Qualified(byte[] data, PrivateKey privKey) throws Exception {
        java.security.Signature sig = java.security.Signature.getInstance("SHA256withRSA");
        sig.initSign(privKey);
        sig.update(data);
        return sig.sign();
    }

    // Test case 11: Other RSA signature algorithms
    public byte[] signWithRsaSha512(byte[] data, PrivateKey privKey) throws Exception {
        Signature sig = Signature.getInstance("SHA512withRSA");
        sig.initSign(privKey);
        sig.update(data);
        return sig.sign();
    }

    // Test case 12: RSA signing via constant propagation
    private static final String RSA_SHA256_ALGO = "SHA256withRSA";
    
    public byte[] signWithRsaConstant(byte[] data, PrivateKey privKey) throws Exception {
        Signature sig = Signature.getInstance(RSA_SHA256_ALGO);
        sig.initSign(privKey);
        sig.update(data);
        return sig.sign();
    }

    // Test case 13: RSA signing via dynamic algorithm (needs manual review)
    public byte[] signWithDynamicAlgorithm(byte[] data, PrivateKey privKey, String algorithm) throws Exception {
        Signature sig = Signature.getInstance(algorithm);
        sig.initSign(privKey);
        sig.update(data);
        return sig.sign();
    }

    // ========== ECDSA SIGNING ==========

    // Test case 14: ECDSA signing with imported Signature and SHA256withECDSA
    public byte[] signWithEcdsaSha256(byte[] data, PrivateKey privKey) throws Exception {
        Signature sig = Signature.getInstance("SHA256withECDSA");
        sig.initSign(privKey);
        sig.update(data);
        return sig.sign();
    }

    // Test case 15: ECDSA signing with fully qualified form
    public byte[] signWithEcdsaSha256Qualified(byte[] data, PrivateKey privKey) throws Exception {
        java.security.Signature sig = java.security.Signature.getInstance("SHA256withECDSA");
        sig.initSign(privKey);
        sig.update(data);
        return sig.sign();
    }

    // Test case 16: Other ECDSA signature algorithms
    public byte[] signWithEcdsaSha512(byte[] data, PrivateKey privKey) throws Exception {
        Signature sig = Signature.getInstance("SHA512withECDSA");
        sig.initSign(privKey);
        sig.update(data);
        return sig.sign();
    }

    // ========== ECDH KEY AGREEMENT ==========

    // Test case 17: ECDH key agreement with imported KeyAgreement
    public byte[] ecdhKeyAgreement(PrivateKey privKey, PublicKey pubKey) throws Exception {
        KeyAgreement ka = KeyAgreement.getInstance("ECDH");
        ka.init(privKey);
        ka.doPhase(pubKey, true);
        return ka.generateSecret();
    }

    // Test case 18: ECDH key agreement with fully qualified form
    public byte[] ecdhKeyAgreementQualified(PrivateKey privKey, PublicKey pubKey) throws Exception {
        javax.crypto.KeyAgreement ka = javax.crypto.KeyAgreement.getInstance("ECDH");
        ka.init(privKey);
        ka.doPhase(pubKey, true);
        return ka.generateSecret();
    }

    // ========== BOUNCYCASTLE PATTERNS ==========

    // Test case 19: BouncyCastle ECC spec (if library is available)
    public void useBouncyCastleEcc() {
        // This would trigger if BouncyCastle is available
        // org.bouncycastle.jce.spec.ECParameterSpec ecSpec = ...
    }
}
