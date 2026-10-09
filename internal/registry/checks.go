package registry

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type RegistrationInput struct {
	Entry    Entry
	Author   string
	Existing map[string]Entry
	Reserved ReservedConfig
	QTIOrg   string
}

func CheckRegistration(ctx context.Context, gh GitHub, in RegistrationInput) error {
	var errs []error

	if err := in.Entry.Validate(); err != nil {
		errs = append(errs, err)
	}

	if in.Entry.Subdomain != "" {
		if _, ok := in.Existing[in.Entry.Subdomain]; ok {
			errs = append(errs, fmt.Errorf("registry: subdomain %q is already registered", in.Entry.Subdomain))
		}
		if in.Reserved.IsReserved(in.Entry.Subdomain) {
			errs = append(errs, fmt.Errorf("registry: subdomain %q is reserved", in.Entry.Subdomain))
		}
	}

	if in.QTIOrg == "" {
		errs = append(errs, errors.New("registry: QTIOrg must be set"))
	}

	if owner, repo, err := ParseRepo(in.Entry.Repo); err == nil && in.QTIOrg != "" {
		member, err := gh.OrgMember(ctx, in.QTIOrg, in.Author)
		if err != nil {
			errs = append(errs, fmt.Errorf("registry: checking %s org membership: %w", in.QTIOrg, err))
		} else if !member {
			hasAccess, err := gh.HasWriteAccess(ctx, owner, repo, in.Author)
			if err != nil {
				errs = append(errs, fmt.Errorf("registry: checking write access to %s/%s: %w", owner, repo, err))
			} else if !hasAccess {
				errs = append(errs, fmt.Errorf("registry: %s does not have write access to %s/%s", in.Author, owner, repo))
			}
		}

		exists, err := gh.PathExists(ctx, owner, repo, in.Entry.Branch, strings.TrimSuffix(in.Entry.Path, "/"))
		if err != nil {
			errs = append(errs, fmt.Errorf("registry: checking %s/%s@%s:%s exists: %w", owner, repo, in.Entry.Branch, in.Entry.Path, err))
		} else if !exists {
			errs = append(errs, fmt.Errorf("registry: %s/%s has no %q at branch %q", owner, repo, in.Entry.Path, in.Entry.Branch))
		}

		for _, v := range in.Entry.Versions {
			vExists, err := gh.PathExists(ctx, owner, repo, v.Ref, strings.TrimSuffix(in.Entry.Path, "/"))
			if err != nil {
				errs = append(errs, fmt.Errorf("registry: checking %s/%s@%s:%s exists (version %q): %w", owner, repo, v.Ref, in.Entry.Path, v.Name, err))
			} else if !vExists {
				errs = append(errs, fmt.Errorf("registry: %s/%s has no %q at ref %q (version %q)", owner, repo, in.Entry.Path, v.Ref, v.Name))
			}
		}
	}

	return errors.Join(errs...)
}

type EditInput struct {
	Existing Entry
	New      Entry
	Author   string
}

func CheckEdit(in EditInput) error {
	var errs []error

	if err := in.New.Validate(); err != nil {
		errs = append(errs, err)
	}
	if in.New.Subdomain != in.Existing.Subdomain {
		errs = append(errs, fmt.Errorf("registry: subdomain cannot be changed by an edit (was %q, now %q); unregister and re-register instead", in.Existing.Subdomain, in.New.Subdomain))
	}

	authorized := false
	for _, m := range in.Existing.Maintainers {
		if strings.EqualFold(m, in.Author) {
			authorized = true
			break
		}
	}
	if !authorized {
		errs = append(errs, fmt.Errorf("registry: %s is not listed in maintainers for %q", in.Author, in.Existing.Subdomain))
	}

	return errors.Join(errs...)
}

type UnregisterInput struct {
	Existing Entry
	Author   string
}

func CheckUnregister(in UnregisterInput) error {
	for _, m := range in.Existing.Maintainers {
		if strings.EqualFold(m, in.Author) {
			return nil
		}
	}
	return fmt.Errorf("registry: %s is not listed in maintainers for %q", in.Author, in.Existing.Subdomain)
}
