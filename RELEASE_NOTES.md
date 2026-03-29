## v0.13.0

This release expands interactive container workflows and introduces the first modern workspace layout, while keeping the tag-driven release process unchanged.

#### Highlights

* Add the new `--workspace` mode with navigator, context, and activity panes plus a compact fallback for smaller terminals. (#267)
* Add interactive container `exec` support from the command menu and command palette. (#264)
* Add interactive container `attach` support for running containers. (#263)
* Add `--theme` / `-T` startup theme selection and in-app theme cycling. (#265)
* Update Bubble Tea and Lip Gloss to `v2.0.2`. (#266)

#### Release prep notes

* Release version should be tagged as `v0.13.0`, based on the new features added since `v0.12.2`.
* Release automation is still triggered by pushing a `v*` tag and handled by GoReleaser via `.github/workflows/release.yml`.
* The current release candidate also includes local stability fixes for workspace stream handling and the `ntcharts/v2` dependency migration.

## v0.6-alpha.1

This version of **dry** is the first one built using Go 1.7, which has resulted in a smaller binary size and maybe in some performance improvements (no measure has been done on this).

The capability to use [termui](https://github.com/gizak/termui) widgets has been added. So far this has been used to add the container menu and to improve the stats screen.

#### Improvements

* Pressing [Enter] on a container now shows a menu with the all the commands that can be executed on a container. Existing keybinds still work, but this change should make it easier to explore what can be done with **dry** on a container. #18
* Improve stats screen to show detailed container information and stats in a nicer way.
* Remove dangling images with [Ctrl+D]. #19
* Container inspect now binded to [I], was on [Enter].

#### Notices

**dry** has been built using Go 1.7 (1.7rc5, the latest beta version available at the time of this writing).

As stated in the 1.7 [release notes](https://tip.golang.org/doc/go1.7#compiler), changes in the compiler toolchain and standard libraries should result in smaller binaries.

The following table shows a comparison of **dry** binary sizes (in bytes) using 1.6.3 and 1.7rc5.
```
os-cpu                1.6.3     1.7       Binary size decrease
dry-darwin-amd64      9666128   7321376   -24,26%
dry-freebsd-amd64     9670081   7350169   -23,99%
dry-linux-amd64       9666625   7333489   -24,14%
dry-windows-amd64     9664000   7298048   -24,48%
dry-darwin-386        7629836   6464384   -15,27%
dry-freebsd-386       7603305   6457457   -15,07%
dry-linux-386         7652591   6473134   -15,41%
dry-windows-386       7690752   6481920   -15,72%
dry-freebsd-arm       7621922   6647331   -12,79%
dry-linux-arm         7613761   6617809   -13,08%
```

So, changing to the **Go 1.7** has resulted in, on average, a 24% decrease in binary sizes for x86-64 architectures. Good job, Go team!
