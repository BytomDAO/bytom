import os

account = os.popen('./bytomcli create-account fdafa')

print account.read()

asset = os.popen('./bytomcli create-asset  fafdafd')

print asset.read()
