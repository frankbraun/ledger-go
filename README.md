## ledger-go

Simple tool that parses a subset of the [Ledger](https://ledger-cli.org)
command-line accounting file format.

The main use case is to enforce that every entry is linked to an invoice file
via metadata annotations.

Invoice files are like this:

```
    ; file: /path/to/invoice.pdf    
```

The integrity of invoice files is ensured with SHA-256 hashes, annotated like
this:

```
    ; sha256: 89bc469fad2dfa90807a236554e6a2c9cb40fce745c50d7c80cc5ae65bfc3cf7
```

Use option `-add-missing-hashes` to add missing SHA-256 automatically.

By default invoice files and entries are a one-to-one mapping. In order to add the same invoice to multiple entries mark it like this:

```
    ; duplicate: true
```