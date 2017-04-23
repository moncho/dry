package wincred

import (
	"syscall"
)

// Get the generic credential with the given name from Windows credential manager
func GetGenericCredential(targetName string) (*GenericCredential, error) {
	cred, err := nativeCredRead(targetName, naCRED_TYPE_GENERIC)
	if cred != nil {
		return &GenericCredential{*cred}, err
	}
	return nil, err
}

// Create a new generic credential with the given name
func NewGenericCredential(targetName string) (result *GenericCredential) {
	result = new(GenericCredential)
	result.TargetName = targetName
	result.Persist = PersistLocalMachine
	return
}

// Persist the credential to Windows credential manager
func (t *GenericCredential) Write() (err error) {
	err = nativeCredWrite(&t.Credential, naCRED_TYPE_GENERIC)
	return
}

// Delete the credential from Windows credential manager
func (t *GenericCredential) Delete() (err error) {
	err = nativeCredDelete(&t.Credential, naCRED_TYPE_GENERIC)
	return
}

// Get the domain password credential with the given target host name
func GetDomainPassword(targetName string) (*DomainPassword, error) {
	cred, err := nativeCredRead(targetName, naCRED_TYPE_DOMAIN_PASSWORD)
	if cred != nil {
		return &DomainPassword{*cred}, err
	}
	return nil, err
}

// Create a new domain password credential used for login to the given target host name
func NewDomainPassword(targetName string) (result *DomainPassword) {
	result = new(DomainPassword)
	result.TargetName = targetName
	result.Persist = PersistLocalMachine
	return
}

// Persist the domain password credential to Windows credential manager
func (t *DomainPassword) Write() (err error) {
	err = nativeCredWrite(&t.Credential, naCRED_TYPE_DOMAIN_PASSWORD)
	return
}

// Delete the domain password credential from Windows credential manager
func (t *DomainPassword) Delete() (err error) {
	err = nativeCredDelete(&t.Credential, naCRED_TYPE_DOMAIN_PASSWORD)
	return
}

// Set the CredentialBlob field of a domain password credential
// using an UTF16 encoded password string
func (t *DomainPassword) SetPassword(pw string) {
	t.CredentialBlob = utf16ToByte(syscall.StringToUTF16(pw))
}

// List the contents of the Credentials store
func List() ([]*Credential, error) {
	creds, err := nativeCredEnumerate("", true)
	if err != nil && err.Error() == naERROR_NOT_FOUND {
		// Ignore ERROR_NOT_FOUND and return an empty list instead
		creds = []*Credential{}
		err = nil
	}
	return creds, err
}
