# What is being tested?

OS login is a ssh mechanism that is enabled when "enable-oslogin=True" is defined
in project metadata.

This test verifies if the image behaves appropriately when
"enable-oslogin" metadata is "True" or not.

Checkes if the following modules are responding correctly:
- nss (getent passwd)
- authorized keys command (/usr/bin/google\_authorized\_keys)
- calls to the metadata server for authorization checks and user lookups.

It also performs end to end tests where:
- Verify a user cannot log into a VM.
- Set IAM permission on a VM for login.
- Log in and verify no sudo.
- Add sudo IAM permission
- Log in again and verify sudo.

# How this test works?

- master-tester: machine who coordinates the tests.
- oslogin-ssh-tester: machine with roles/compute.osLogin used by the master to test login of a user with no sudo permissions.
- osadminlogin-ssh-tester: machine with roles/compute.osAdminLogin used by the master to test login of a user with sudo permissions.
- oslogin-ssh-testee: machine instanciated with the image being tested.

# Setup

Create two service accounts with the following roles:

-   `daisy-oslogin`:
     roles/compute.osLogin
     roles/iam.serviceAccountUser
     roles/storage.objectViewer

-   `daisy-osadminlogin`:
     roles/compute.osAdminLogin
     roles/iam.serviceAccountUser
     roles/storage.objectViewer

You can use the following commands

    gcloud iam service-accounts create daisy-oslogin
    gcloud projects add-iam-policy-binding ${PROJECT} --member='serviceAccount:daisy-oslogin@${PROJECT}.iam.gserviceaccount.com' --role='roles/compute.osLogin'
    gcloud projects add-iam-policy-binding ${PROJECT} --member='serviceAccount:daisy-oslogin@${PROJECT}.iam.gserviceaccount.com' --role='roles/iam.serviceAccountUser'
    gcloud projects add-iam-policy-binding ${PROJECT} --member='serviceAccount:daisy-oslogin@${PROJECT}.iam.gserviceaccount.com' --role='roles/storage.objectViewer'

    gcloud iam service-accounts create daisy-osadminlogin
    gcloud projects add-iam-policy-binding ${PROJECT} --member='serviceAccount:daisy-osadminlogin@${PROJECT}.iam.gserviceaccount.com' --role='roles/compute.osAdminLogin'
    gcloud projects add-iam-policy-binding ${PROJECT} --member='serviceAccount:daisy-osadminlogin@${PROJECT}.iam.gserviceaccount.com' --role='roles/iam.serviceAccountUser'
    gcloud projects add-iam-policy-binding ${PROJECT} --member='serviceAccount:daisy-osadminlogin@${PROJECT}.iam.gserviceaccount.com' --role='roles/storage.objectViewer'

