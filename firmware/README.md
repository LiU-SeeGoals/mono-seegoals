# Firmware
Here is all the firmware used for the robots and basestation in the SeeGoals project.

Further details can be found within the subfolders `README.md`, but here follows a quick description.

```
./basestation/      - Basestation firmware
./datasheets/       - Datasheets specific for our hardware.
./robot/            - Robot firmware
./shared/           - Firmware code shared between the two
./.clang-format     - Auto-format rules for the `clangd` LSP server
./.clangd           - Specific settings for the `clangd` LSP server
./.editorconfig     - Format rules for editors respecting a editor config file
./CMakeLists.txt    - The main CMakeLists file, both `basestation` and `robot` have `*.cmake` files with more specific configurations.
```

## Compiling and flashing
See the `sg-fw` command.

TODO:
- Manually?

## Debugging
See `sg-fw serial` command.

TODO:
- GDB?

## Clangd
We use `clangd` as our LSP server of choice.

TODO:
- How to setup? Vim/VS code
