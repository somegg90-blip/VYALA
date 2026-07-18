from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.hazmat.primitives.asymmetric.ec import ECDSA
from cryptography.hazmat.primitives import hashes

def sign_data(key, msg):
    return key.sign(msg, ECDSA(hashes.SHA256()))
