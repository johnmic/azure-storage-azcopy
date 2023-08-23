#!/bin/bash

# This script is used to run e2e tests on Linux VMs.

# verify the script is running as root and exit if not
if [ "$EUID" -ne 0 ]
  then echo "Please run as root"
  exit
fi

source /etc/environment

# If AZCOPY_E2E_ACCOUNT_NAME is not set in the user environment variables, the script will prompt for the value and set it in the user local profile.
if [ -z "$AZCOPY_E2E_ACCOUNT_NAME" ]
then
    echo -n 'Azure storage account name (with hierarchical namespace disabled):'
    read accountName
    echo 'AZCOPY_E2E_ACCOUNT_NAME="'$accountName'"' >> /etc/environment
    echo 'AZCOPY_E2E_CLASSIC_ACCOUNT_NAME="'$accountName'"' >> /etc/environment
fi

# If AZCOPY_E2E_ACCOUNT_KEY is not set in the user environment variables, the script will prompt for the value and set it in the user local profile.
if [ -z "$AZCOPY_E2E_ACCOUNT_KEY" ]
then
    echo -n 'Azure storage account key (with hierarchical namespace disabled):'
    read accountKey
    echo 'AZCOPY_E2E_ACCOUNT_KEY="'$accountKey'"' >> /etc/environment
    echo 'AZCOPY_E2E_CLASSIC_ACCOUNT_KEY="'$accountKey'"' >> /etc/environment
fi

# If AZCOPY_E2E_ACCOUNT_NAME_HNS is not set in the user environment variables, the script will prompt for the value and set it in the user local profile.
if [ -z "$AZCOPY_E2E_ACCOUNT_NAME_HNS" ]
then
    echo -n 'Azure storage account hns name (with hierarchical namespace enabled):'
    read hnsAccountName
    echo 'AZCOPY_E2E_ACCOUNT_NAME_HNS="'$hnsAccountName'"' >> /etc/environment
fi

# If AZCOPY_E2E_ACCOUNT_KEY_HNS is not set in the user environment variables, the script will prompt for the value and set it in the user local profile.
if [ -z "$AZCOPY_E2E_ACCOUNT_KEY_HNS" ]
then
    echo -n 'Azure storage account hns key (with hierarchical namespace enabled):'
    read hnsAccountKey
    echo 'AZCOPY_E2E_ACCOUNT_KEY_HNS="'$hnsAccountKey'"' >> /etc/environment
fi

# If AZCOPY_E2E_SMB_MOUNT_HOST is not set in the user environment variables, the script will prompt for the value and set it in the user local profile.
if [ -z "$AZCOPY_E2E_SMB_MOUNT_HOST" ]
then
    echo -n 'SMB mount host (DNS / IP):'
    read smbHost
    echo 'AZCOPY_E2E_SMB_MOUNT_HOST="'$smbHost'"' >> /etc/environment
fi

# If AZCOPY_E2E_SMB_MOUNT_SHARE is not set in the user environment variables, the script will prompt for the value and set it in the user local profile.
if [ -z "$AZCOPY_E2E_SMB_MOUNT_SHARE" ]
then
    echo -n 'SMB mount share name:'
    read smbRemotePath
    echo 'AZCOPY_E2E_SMB_MOUNT_SHARE="'$smbRemotePath'"' >> /etc/environment
fi


if [ -z "$SMB_USER_NAME" ]
then
    echo -n 'SMB user name:'
    read smbUserName
    echo 'SMB_USER_NAME="'$smbUserName'"'  >> /etc/environment
fi

if [ -z "$SMB_PASSWORD" ]
then
    echo -n 'SMB password:'
    read smbPassword
    echo 'SMB_PASSWORD="'$smbPassword'"'  >> /etc/environment
fi

# If AZCOPY_E2E_TENANT_ID is not set in the user environment variables, the script will prompt for the value and set it in the user local profile.
if [ -z "$AZCOPY_E2E_TENANT_ID" ]
then
    echo -n 'Tenant Id:'
    read tenantId
    echo 'AZCOPY_E2E_TENANT_ID="'$tenantId'"' >> /etc/environment
fi

# If AZCOPY_E2E_APPLICATION_ID is not set in the user environment variables, the script will prompt for the value and set it in the user local profile.
if [ -z "$AZCOPY_E2E_APPLICATION_ID" ]
then
    echo -n 'App registration Application Id:'
    read applicationId
    echo 'AZCOPY_E2E_APPLICATION_ID="'$applicationId'"' >> /etc/environment
fi

# If AZCOPY_E2E_CLIENT_SECRET is not set in the user environment variables, the script will prompt for the value and set it in the user local profile.
if [ -z "$AZCOPY_E2E_CLIENT_SECRET" ]
then
    echo -n 'App registration secret value:'
    read secretValue
    echo 'AZCOPY_E2E_CLIENT_SECRET="'$secretValue'"' >> /etc/environment
fi

if [ -z "$CPK_ENCRYPTION_KEY" ]
then
    encrypitonKey=$(uuidgen)
    echo 'CPK_ENCRYPTION_KEY="'$encrypitonKey'"'  >> /etc/environment
fi

if [ -z "$CPK_ENCRYPTION_KEY_SHA256" ]
then
    sha256Hash=$(echo -n $encrypitonKey | shasum -a 256)
    echo 'CPK_ENCRYPTION_KEY_SHA256="'$sha256Hash'"'  >> /etc/environment
fi

echo '-------------------Mount SMB---------------------------'
source /etc/environment
sudo umount /mnt/AzCopyE2ESMB
sudo rm -rf /mnt/AzCopyE2ESMB
sudo mkdir -p /mnt/AzCopyE2ESMB
sudo mount -t cifs //$AZCOPY_E2E_SMB_MOUNT_HOST/$AZCOPY_E2E_SMB_MOUNT_SHARE /mnt/AzCopyE2ESMB -o rw,vers=3.0,username=$SMB_USER_NAME,password=$SMB_PASSWORD
if [ $? -ne 0 ]; then
    echo "Failed to mount SMB share"
    exit 1
fi

echo '------------------Build azCopy-------------------------'

GOARCH=amd64 GOOS=linux go build -o azcopy_linux_amd64
if [ -z "$AZCOPY_E2E_EXECUTABLE_PATH" ]
then
    echo 'AZCOPY_E2E_EXECUTABLE_PATH='$(pwd)'/azcopy_linux_amd64'  >> /etc/environment
fi

echo '------------------Running E2E Tests -------------------'
source /etc/environment
go test -v -timeout 60m -race -short -cover ./e2etest
echo '--------------Running E2E Tests Finished---------------'
