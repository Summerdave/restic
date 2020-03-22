package main

import (
	"context"
	"fmt"

	"github.com/restic/restic/internal/errors"
	"github.com/restic/restic/internal/index"
	"github.com/restic/restic/internal/restic"

	"github.com/spf13/cobra"
)

var cmdReferencedPacks = &cobra.Command{
	Use:   "referenced-packs snapshot ID [snapshot ID]",
	Short: "List packs referenced by one or more snapshots",
	Long: `
The referenced-packs command lists all packs that are referenced by the
given list of snapshots.
	
EXIT STATUS
===========

Exit status is 0 if the command was successful, and non-zero if there was any error.
	`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReferencedPacks(globalOptions, args)
	},
}

// type ReferencedPacksOptions struct {
// }

// var referencedPacksOptions ReferencedPacksOptions

func init() {
	cmdRoot.AddCommand(cmdReferencedPacks)
}

func runReferencedPacks(gopts GlobalOptions, args []string) error {
	gopts.Quiet = true
	repo, err := OpenRepository(gopts)
	ctx, _ := context.WithCancel(gopts.ctx)
	{
		err := repo.LoadIndex(ctx)
		if err != nil {
			return err
		}
	}
	progress := restic.NewProgress()
	idx, err := index.Load(ctx, repo, progress)
	if err != nil {
		return err
	}
	var snapshots restic.Snapshots

	for sn := range FindFilteredSnapshots(ctx, repo, nil, nil, nil, args) {
		snapshots = append(snapshots, sn)
	}

	blobs := 0
	for _, pack := range idx.Packs {
		blobs += len(pack.Entries)
	}

	usedBlobs := restic.NewBlobSet()
	seenBlobs := restic.NewBlobSet()

	for _, sn := range snapshots {
		err = restic.FindUsedBlobs(ctx, repo, *sn.Tree, usedBlobs, seenBlobs)
		if err != nil {
			if repo.Backend().IsNotExist(err) {
				return errors.Fatal("unable to load a tree from the repo: " + err.Error())
			}
			return err
		}
	}

	packIds := map[restic.ID]struct{}{}

	for _, pack := range idx.Packs {
		for _, blob := range pack.Entries {
			h := restic.BlobHandle{ID: blob.ID, Type: blob.Type}
			if usedBlobs.Has(h) {
				packIds[pack.ID] = struct{}{}
			}
		}
	}

	for packId := range packIds {
		fmt.Printf("%v\n", packId)
	}

	return err
}
