# TODOs for FSG

## Very bad

- proper permissions for everything
- fix project edit dialog
- add pagination or infiniscroll to project tags

## Bad

- tag table broken when no tags available in backend

## Not so bad

- delete cameras
- tag ui broken when tag description too long


# Permission TODOs:
## Auth required:

### cameras
ok

### image_tag_assignments
CREATE: user is admin || projectAdmin of project || owner of image
READ:   user is admin || assigned to project
UPDATE: user is admin || projectAdmin of project || owner of image
DELETE: user is admin || projectAdmin of project || owner of image

### image_tags
!CREATE: user is admin || projectAdmin of project || (projectEditor of project && image_tags type is custom)
READ:   user is admin || assigned to project
!UPDATE: user is admin || projectAdmin of project
!DELETE: user is admin || projectAdmin of project

### images
READ:  user is admin || assigned to project
!CREATE: user is admin || projectAdmin of project || projectEditor of project
!UPDATE: user is admin || projectAdmin of project || owner of image
!DELETE: user is admin || projectAdmin of project || owner of image

### project_assignments
CREATE: user is admin || projectAdmin of project being assigned
READ:   user is admin || projectAdmin of project being assigned
UPDATE: user is admin || projectAdmin of project being assigned
DELETE: user is admin || projectAdmin of project being assigned

### project
UPDATE: user is admin || projectAdmin of project

### roles
ok

### time_offsets
ok

### uploads
CREATE: user is admin || projectAdmin of project || projectEditor of project
READ:   user is admin || assigned to project
UPDATE: user is admin || projectAdmin of project || owner of upload
DELETE: user is admin || projectAdmin of project || owner of upload

