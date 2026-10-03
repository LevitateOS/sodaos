# Optional laptop key enrollment

Import one laptop SSH public key when no operator key is installed yet. This
path is optional: skip it when the operator's key is already in place.

## Before starting

- You hold interactive root terminal access to the appliance (local console or
  remote SSH) and know the native root password.
- Your laptop can reach a private host address on the same network. Only a
  live private IPv4 address is admitted; an interface address alone does not
  prove a client route.

## Arm the import window

On the host, as root:

```sh
soda-install enroll-key
```

1. Choose the numbered private interface/address your laptop can actually
   reach. Answer `back` or `cancel` to exit without opening anything.
2. Read the displayed native Ed25519 host fingerprint. It is derived from the
   actual private host key; you will confirm the same fingerprint on the
   laptop before sending anything.
3. Type `ARM KEY IMPORT` to open the window. A temporary password-only
   key-import listener serves the selected address on port `22222` for at
   most five minutes. It uses OpenSSH's native password verification;
   additional PAM policies are not inherited, and ordinary SSH policy is
   unchanged.

## Send the laptop key

From the laptop, with the host fingerprint verified through
`StrictHostKeyChecking=ask`:

```sh
ssh -T -p 22222 -o StrictHostKeyChecking=ask -o PreferredAuthentications=password -o PubkeyAuthentication=no root@HOST_ADDRESS < ~/.ssh/id_ed25519.pub
```

Exactly one bounded public-key file is accepted. Press Enter or Ctrl-C on the
host console to close the window early.

## If the window closes first

The window expires after five minutes, aborts when the selected interface
address changes, and never replaces an existing enrollment service or state:
finish or inspect the earlier attempt locally before arming another. When a
close cannot be confirmed, verify native key access locally before importing
again; a key may already have been imported.

After a successful import, continue with
[operator setup](../public/20-Deploy/25-operator-setup.md) using key
authentication.
