package wincred

import (
	"time"
)

type CredentialPersistence uint32

const (
	PersistSession      CredentialPersistence = 0x1
	PersistLocalMachine CredentialPersistence = 0x2
	PersistEnterprise   CredentialPersistence = 0x3
)

type CredentialAttribute struct {
	Keyword string
	Value   []byte
}

type Credential struct {
	TargetName     string
	Comment        string
	LastWritten    time.Time
	CredentialBlob []byte
	Attributes     []CredentialAttribute
	TargetAlias    string
	UserName       string
	Persist        CredentialPersistence
}

type GenericCredential struct {
	Credential
}

type DomainPassword struct {
	Credential
}
