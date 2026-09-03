package datastore

import (
	"context"

	"silvatek.uk/trustedassertions/internal/auth"
)

func AddPasskey(ctx context.Context, userID string, pk auth.Passkey) error {
	return AddPasskeyTo(ctx, ActiveDataStore, userID, pk)
}

func RemovePasskey(ctx context.Context, userID string, credentialID []byte) error {
	return RemovePasskeyFrom(ctx, ActiveDataStore, userID, credentialID)
}

func AddPasskeyTo(ctx context.Context, store DataStore, userID string, pk auth.Passkey) error {
	user, err := store.FetchUser(ctx, userID)
	if err != nil {
		return err
	}
	if err := user.AddPasskey(pk); err != nil {
		return err
	}
	store.StoreUser(ctx, user)
	return nil
}

func RemovePasskeyFrom(ctx context.Context, store DataStore, userID string, credentialID []byte) error {
	user, err := store.FetchUser(ctx, userID)
	if err != nil {
		return err
	}
	if err := user.RemovePasskey(credentialID); err != nil {
		return err
	}
	store.StoreUser(ctx, user)
	return nil
}
