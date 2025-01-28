# NDNCERT CLI Reference

The NDNCERT CLI tool can be used to interact with an NDNCERT CA running in an NDN network.

This example illustrates how to get a certificate from the NDN Testbed CA.

## Getting the CA Certificate

To begin, you need to get the root CA certificate out of band. The most recent certificates for the NDN Testbed can be obtained [here](https://named-data.net/ndn-testbed/).

If the certificate is in base64, you will need to convert it to TLV or PEM format.

```sh
cat testbed.root.base64 | base64 -d | ndnd sec pem-encode > testbed.root.pem
```

## Requesting a certificate

In this example, we will use the email challenge to request a certificate from the NDN Testbed CA.

Note that the specified output filename does not include an extension.
In this case, a new key will be created since no key is specified with `-k`.
The key will then be written to `alice.key` and the issued certificate to `alice.cert`.

```sh
ndnd certcli -o alice testbed.root.pem
```

The CLI will now prompt you for challenge type. Select the `email` challenge.

```text
Please choose a challenge type:
  1. email
  2. pin
Choice: 1
```

You will next be prompted for the email address to use for the challenge.

```text
Enter your email address: alice@named-data.net
```

The CLI will now attempt to request a certificate from the CA.

```text
================ CA Profile ===============
NDN Testbed NDNCERT CA (Demo)
Name: /ndn
Max Validity: 360h0m0s
Challenges: [email]
===========================================

Challenge Status: need-code
Enter the code sent to your email address:
```

You will receive an email with a code to enter.
On successful verification, the certificate will be issued,
and the CLI will write the certificate to the specified output file.
