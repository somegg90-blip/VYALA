# auth.py - user authentication module (should trigger HIGH severity)
from cryptography.hazmat.primitives.asymmetric import rsa, ec
from cryptography.hazmat.primitives.asymmetric.ec import SECP256R1

# RSA key generation - should be caught
private_rsa = rsa.generate_private_key(public_exponent=65537, key_size=2048)

# Classical ECC key generation - should be caught
private_ec = ec.generate_private_key(SECP256R1())

# Another RSA call on a different line
another_rsa = rsa.generate_private_key(65537, 4096)

# ECDSA signing (classical) - should be caught
def sign_data(data):
    from cryptography.hazmat.primitives import hashes
    from cryptography.hazmat.primitives.asymmetric import padding
    # This is just a placeholder; the actual call would be caught as well.
    pass
# test force-push dedup
