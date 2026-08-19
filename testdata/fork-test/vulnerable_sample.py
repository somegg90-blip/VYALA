from cryptography.hazmat.primitives.asymmetric import rsa, ec, padding
from cryptography.hazmat.primitives.serialization import load_pem_private_key

# RSA key generation — should trigger rsa_keygen (HIGH, in a payment/ or auth/ path it'd be higher)
private_key = rsa.generate_private_key(public_exponent=65537, key_size=2048)

# Classical ECC usage — should trigger classical_ecc
ec_key = ec.generate_private_key(ec.SECP256R1())

# RSA signing — should trigger rsa_signing_usage
message = b"test payload"
signature = private_key.sign(message, padding.PKCS1v15(), None)

# Key loading — should trigger key_loading (INFO/low)
with open("some_key.pem", "rb") as f:
    loaded_key = load_pem_private_key(f.read(), password=None)
