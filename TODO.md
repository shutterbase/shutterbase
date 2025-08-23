# TODOs for FSG

## Very bad

- hotkeys blocking inputs in text fields

## Bad

- close tagging dialog when clicking outside of it
- filter for upload with query parameter, etc
- transaction when removing tag (concurrently removig tag and switching to next image causes issues; address image by ID instead of slice item; make sure to address image ID, not current image)

## Not so bad

- delete cameras
- add pagination or infiniscroll to project tags
- tag ui broken when tag description too long
- applied tag shows up locally as "manual" even though it might be "custom"


# 2025 TODOs

## default tag "review" - only to be removed by project admin

## customizable hotkey functions

- allow users to define their own hotkeys for common actions
- provide a default set of hotkeys

### list of hot-keyable operations
- specific tag assignment
- specific tag removal
- grid / detail switch
- open tagging window
- repeat
- up / down / left / right
- last applied tag (n-1)
- n-2 - n-5 applied tag

## tag coupling
- when tag A is applied, also tag B is applied
- cascade tag removal: when tag A is removed, also tag B is removed

## downloader 
- keep AND concatenated whitelist
- add OR concatenated blacklist