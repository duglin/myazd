# azx

## What is it?

A proof-of-concept tool that explores what it would look like to have an
Infrastructure-as-Code (IaC) editor that had an `az`-like user experience.
The goal is to see how far people go using an `az` syntax to configure
their IaC w/o the need to touch IaC directly.

However, this also goes with the assumption that evetually some people will
need to drop down into the IaC so some design points:
- only put stuff in the IaC that the user explicitly provided. This means
  that the IaC should be (sort of) readable by the user since while the
  syntax while be new to them, the values and properties mentioned are all
  ones that the user has expressed an interest in. The learning curve is
  reduced down to "syntax", not "semantics". No wall of ARM/JSON to read
  just to find the properties of interest.
- similarly, defaults are not added to the IaC files. Defaults will be applied
  by the RPs or before the request is sent to the server.
- as the user runs the `az`-like CLI, all of the IaC changes are immediately
  saved to disk and can be checked into git. This means the user can view
  the CLI differently based on their perspective:
  - an IaC editor
  - a tool that saves a history of their commands so they don't need to
    remember everything they did to repeat it later
  - a tool the does imperative and declarative management at the same time,
    if that what's the user chooses - meaning, each az-like command will
	edit the local IaC files but can (optionally) also apply the changes to
	Azure at the same time.
- all output defaults to human readble output, including doing "diff"s.
  However, the user should be able to ask for the underlying bits if they
  want to see what's going on. This means, seeing the current IaC files
  as well as seeing what is sent to Azure (which means the current IaC plus
  any tweaking (e.g. defaults) that might need to be done).
- No new IaC files or formats are created. People should be able to use
  the saved IaC files outside of this tool. The tool isn't meant to be a
  new UX that people are forced to use, rather it is just there to help
  people create/manage IaC files.
- this means that people can, and should if necessarily, switch between
  the CLI and touching the IaC files directly based on their needs or
  preferred development style. In other words, touching the IaC files
  outside of the CLI doesn't break anything.
- the IaC files currently use the Azure REST JSON, not ARM or Bicep, simply
  for convinience - no need to convert anything before sending it to the 
  server. It also, keeps things really simple if we don't get into the
  complexity of Bicep.


## Playing

### Build

```
$ make
```

will build a mac, linux and windows version of the CLI tool.

### Running

```
$ azx
```

is the CLI tool and will provide help text as necessarily

```
$ azx init
```
will create a new project in the current directory. Really, this just means
it'll creat a hidden directory to store all of the IaC files.

### demos

There are a few demos in here (`demo1`, `demo2`). They assume some stuff
is already setup in advance so read the comments to see how to run each one.
There might be some stuff specific to my setup that I missed.

However, look at [demo1-output.docx](demo1-output.docx) to see a sample
output of `demo1` so you can see how it looks/feels w/o needing to run it
yourself.

The demos are real/live demos, but use a script to do the typing of the
commands for you. Use the spacebar to step forward to the next command.


## Thoughts

Ping me with comments or suggestions!
