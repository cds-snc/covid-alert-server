# Commit Message Strategy
Covid Alert Server uses the Angular commit convention for implementing human/machine readable commits.

We use these commits to generate release documentation.

## Reference Documentation
It is recommended that you read the following documentation:

- **Angular Commit Guidelines**: https://github.com/angular/angular/blob/22b96b9/CONTRIBUTING.md#commit
- **Conventional Commits**: https://www.conventionalcommits.org/en/v1.0.0/
- **Semantic Versioning**: https://semver.org/

## Converting Commits to Semantic Versionsing

### Major
Any commit message with a  ! appended at the end of the commit type  
or containing a footer with Breaking Change is considered a breaking change and will be converted to Major.


### Minor
-feat:
### Patch
- fix
### None
- style:
- refactor:
- chore:
- build:
- ci:
- docs:
- perf:
- test:

## Scope
Each commit can also have a scope. 

The following is a non-exhaustive list of things to scope to, if adding a new scope please add to the list.

- developer_environment
- db
- retrieval
- submission
- scaffolding
- scripts
- release | *fix, ci, chore, docs, test only*
- monitoring

