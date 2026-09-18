package utils

import (
	"khairul169/garage-webui/schema"
	"path/filepath"
	"sync"
	"testing"
)

func TestLastAdminProtection(t *testing.T) {
	for _, operation := range []string{"demote", "delete"} {
		t.Run(operation, func(t *testing.T) {
			store := &UserStore{path: filepath.Join(t.TempDir(), "users.json")}
			for _, id := range []string{"one", "two"} {
				if _, err := store.Create(schema.User{ID: id, Username: id, Role: schema.RoleAdmin}); err != nil {
					t.Fatal(err)
				}
			}
			var wg sync.WaitGroup
			errors := make(chan error, 2)
			for _, id := range []string{"one", "two"} {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if operation == "delete" {
						errors <- store.Delete(id)
					} else {
						_, err := store.Update(id, func(u *schema.User) error { u.Role = schema.RoleViewer; return nil })
						errors <- err
					}
				}()
			}
			wg.Wait()
			close(errors)
			failures := 0
			for err := range errors {
				if err != nil {
					failures++
				}
			}
			if failures != 1 || store.CountByRole(schema.RoleAdmin) != 1 {
				t.Fatal("concurrent changes removed the last admin")
			}
		})
	}
}

func TestOnlyOneInitialAdmin(t *testing.T) {
	store := &UserStore{path: filepath.Join(t.TempDir(), "users.json")}
	var wg sync.WaitGroup
	for _, name := range []string{"first", "second"} {
		wg.Add(1)
		go func() { defer wg.Done(); store.CreateInitialAdmin(schema.User{Username: name}) }()
	}
	wg.Wait()
	if store.Count() != 1 || store.CountByRole(schema.RoleAdmin) != 1 {
		t.Fatal("registration created more than one initial admin")
	}
}

func TestOnlyCurrentRolesAccepted(t *testing.T) {
	store := &UserStore{path: filepath.Join(t.TempDir(), "users.json")}
	for _, role := range []schema.Role{"owner", "developer", "unknown", ""} {
		if _, err := store.Create(schema.User{Username: string(role), Role: role}); err == nil {
			t.Fatalf("accepted invalid role %q", role)
		}
	}
}
