// Package snapshot defines fixed generic file trees from explicitly supplied
// values. It performs no acquisition, storage, repository or workspace operations.
// A Tree describes only those values; it makes no claim about a live repository.
//
// Paths retain their exact UTF-8 spelling. NewPath validates a portable,
// slash-separated repository-relative representation without cleaning, folding or
// rewriting it. Trees reject duplicate paths, file/directory conflicts and Unicode
// simple-case-fold aliases, including inconsistent directory spelling. Unicode
// normalization is not performed: distinct normalization forms remain distinct.
// Adapters must separately check target filesystem representability and aliases.
//
// Only regular files, with or without the executable flag, are supported.
// Directories are implicit; empty directories, symlinks, submodules, ownership,
// timestamps and other physical metadata are not represented. Adapters must reject
// unsupported physical entries rather than silently omit or dereference them.
//
// ContentID is SHA-256 of the exact bytes. TreeID uses a separate v1 domain and
// binds sorted exact paths, portable modes and content identities. It is not a
// canonical-model digest, Git object identity or mutable workspace identifier.
// Empty and nil byte inputs describe the same empty content. The zero Tree is the
// empty tree. Zero Paths and Entries are invalid and constructors reject them.
// Fixed values expose no mutable byte, slice or map aliases; returned slices are
// caller-owned copies. Concurrent reads of fixed values need no synchronization.
package snapshot
