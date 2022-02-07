// @ts-ignore
import {prisma, authBlock, authedUserInGroups} from "../util";
// @ts-ignore
import {ApolloError, ForbiddenError} from "apollo-server-errors";


export async function getPicture(parent, args, context, info) {
    const id = args.id;
    const picture = await prisma.picture.findUnique({where: {id}});
    if(!picture) {
        throw new ApolloError(`No Picture with ID '${id}' found`, "NOT_FOUND")
    }
    return picture;
}


export async function getPictures(parent, args, context, info) {
    const filter:any = args.filter;
    const search:string = args.search;
    const sort: any = args.sort;
    let where: any = {};

    if(filter) {
        where['AND'] = [];
        // Filter for width range
        if(filter.widthRange) {
            if(filter.widthRange.gte) {
                where['AND'].push({width: {gte: filter.widthRange.gte}});
            }
            if(filter.widthRange.lte) {
                where['AND'].push({width: {lte: filter.widthRange.lte}});
            }
        }
        // Filter for width match with 'contains' modifier
        if(filter.width) where['AND'].push({width: {contains: filter.width}});
        // Filter for height range
        if(filter.heightRange) {
            if(filter.heightRange.gte) {
                where['AND'].push({height: {gte: filter.heightRange.gte}});
            }
            if(filter.heightRange.lte) {
                where['AND'].push({height: {lte: filter.heightRange.lte}});
            }
        }
        // Filter for height match with 'contains' modifier
        if(filter.height) where['AND'].push({height: {contains: filter.height}});
        // Filter for size range
        if(filter.sizeRange) {
            if(filter.sizeRange.gte) {
                where['AND'].push({size: {gte: filter.sizeRange.gte}});
            }
            if(filter.sizeRange.lte) {
                where['AND'].push({size: {lte: filter.sizeRange.lte}});
            }
        }
        // Filter for size match with 'contains' modifier
        if(filter.size) where['AND'].push({size: {contains: filter.size}});
        // Filter for rating range
        if(filter.ratingRange) {
            if(filter.ratingRange.gte) {
                where['AND'].push({rating: {gte: filter.ratingRange.gte}});
            }
            if(filter.ratingRange.lte) {
                where['AND'].push({rating: {lte: filter.ratingRange.lte}});
            }
        }
        // Filter for rating match with 'contains' modifier
        if(filter.rating) where['AND'].push({rating: {contains: filter.rating}});
        // Filter for createdDate range
        if(filter.createdDateRange) {
            if(filter.createdDateRange.gte) {
                where['AND'].push({createdDate: {gte: filter.createdDateRange.gte}});
            }
            if(filter.createdDateRange.lte) {
                where['AND'].push({createdDate: {lte: filter.createdDateRange.lte}});
            }
        }
        // Filter for createdDate match with 'contains' modifier
        if(filter.createdDate) where['AND'].push({createdDate: {contains: filter.createdDate}});
        // Filter for modifiedDate range
        if(filter.modifiedDateRange) {
            if(filter.modifiedDateRange.gte) {
                where['AND'].push({modifiedDate: {gte: filter.modifiedDateRange.gte}});
            }
            if(filter.modifiedDateRange.lte) {
                where['AND'].push({modifiedDate: {lte: filter.modifiedDateRange.lte}});
            }
        }
        // Filter for modifiedDate match with 'contains' modifier
        if(filter.modifiedDate) where['AND'].push({modifiedDate: {contains: filter.modifiedDate}});
    }

    if(search) {
        where['OR'] = [
            {name: {contains: search}},
            {width: {contains: search}},
            {height: {contains: search}},
            {fileFormat: {contains: search}},
            {size: {contains: search}},
            {rating: {contains: search}},
            {fileName: {contains: search}},
            {originalFileName: {contains: search}},
        ]
    }

    const limit: number = parseInt(args.limit) || 100
    const offset: number = parseInt(args.offset) || 0

    let orderBy: any = [];
    if(sort) {
        for(const key in sort) {
            let sort = {}
            sort[key] = args.sort[key].toLowerCase();
            orderBy.push(sort);
        }
    }

    const results = await prisma.$transaction([
        prisma.picture.count({where}),
        prisma.picture.findMany({
            where,
            orderBy,
            take: limit,
            skip: offset
        })
    ])
    const total = results[0];
    const pictures = results[1];

    return {
        total,
        pictures
    };
}


export async function getUserForPicture(parent, args, context, info) {
    
    const id = parent.id;

    const picture = await prisma.picture.findUnique({where: {id}, include: {user: true}})

    if(picture !== null) {
        return picture.user;
    }
    else {
        return null;
    }
}

export async function getTagsForPicture(parent, args, context, info) {
    
    const id = parent.id;


    const filter:any = args.filter;
    const search:string = args.search;
    const sort: any = args.sort;
    let where: any = {};

    if(filter) {
        where['AND'] = [];
        if(filter.name) where['AND'].push({name: {contains: filter.name}});
    }

    if(search) {
        where['OR'] = [
            {name: {contains: search}},
            {description: {contains: search}},
        ]
    }

    const limit: number = parseInt(args.limit) || 100
    const offset: number = parseInt(args.offset) || 0

    let orderBy: any = [];
    if(sort) {
        for(const key in sort) {
            let sort = {}
            sort[key] = args.sort[key].toLowerCase();
            orderBy.push(sort);
        }
    }

    const picture = await prisma.picture.findUnique({
        where: {id},
        include: {
            tags: { take: limit, skip: offset, where, orderBy}
    }});

    if(picture !== null) {
        return picture.tags;
    }
    else {
        return [];
    }
}

export async function getPreviewForPicture(parent, args, context, info) {
    
    const id = parent.id;

    const picture = await prisma.picture.findUnique({where: {id}, include: {preview: true}})

    if(picture !== null) {
        return picture.preview;
    }
    else {
        return null;
    }
}

export async function getProfilePictureUserForPicture(parent, args, context, info) {
    
    const id = parent.id;

    const picture = await prisma.picture.findUnique({where: {id}, include: {profilePictureUser: true}})

    if(picture !== null) {
        return picture.profilePictureUser;
    }
    else {
        return null;
    }
}

export async function getCollectionsForPicture(parent, args, context, info) {
    
    const id = parent.id;


    const filter:any = args.filter;
    const search:string = args.search;
    const sort: any = args.sort;
    let where: any = {};

    if(filter) {
        where['AND'] = [];
    }

    if(search) {
        where['OR'] = [
            {name: {contains: search}},
            {description: {contains: search}},
        ]
    }

    const limit: number = parseInt(args.limit) || 100
    const offset: number = parseInt(args.offset) || 0

    let orderBy: any = [];
    if(sort) {
        for(const key in sort) {
            let sort = {}
            sort[key] = args.sort[key].toLowerCase();
            orderBy.push(sort);
        }
    }

    const picture = await prisma.picture.findUnique({
        where: {id},
        include: {
            collections: { take: limit, skip: offset, where, orderBy}
    }});

    if(picture !== null) {
        return picture.collections;
    }
    else {
        return [];
    }
}



export async function createPicture(parent, args, context, info) {
    authBlock(context);
    const authorizedGroups = ['picture_creator'];
    if(!authedUserInGroups(context, authorizedGroups)) {
        throw new ForbiddenError(`User is not authorized to perform this action. Authorized groups are ${authorizedGroups}`)
    }

    const data:any = {}
    if('key' in args.inputs) data['key'] = args.inputs['key']
    if('name' in args.inputs) data['name'] = args.inputs['name']
    if('width' in args.inputs) data['width'] = args.inputs['width']
    if('height' in args.inputs) data['height'] = args.inputs['height']
    if('fileFormat' in args.inputs) data['fileFormat'] = args.inputs['fileFormat']
    if('size' in args.inputs) data['size'] = args.inputs['size']
    if('rating' in args.inputs) data['rating'] = args.inputs['rating']
    if('fileName' in args.inputs) data['fileName'] = args.inputs['fileName']
    if('originalFileName' in args.inputs) data['originalFileName'] = args.inputs['originalFileName']
    if('viwes' in args.inputs) data['viwes'] = args.inputs['viwes']
    if('exif' in args.inputs) data['exif'] = args.inputs['exif']
    if('userId' in args.inputs) data['userId'] = args.inputs['userId']
    if('user' in args.inputs) {
        data['user'] = {};
        data['user']['connect'] = { id: args.inputs['user'] };
    }
    if('tags' in args.inputs) {
        data['tags'] = {};
        data['tags']['connect'] = args.inputs['tags'].map((e:number) => { return { id: e } });
    }
    if('preview' in args.inputs) {
        data['preview'] = {};
        data['preview']['connect'] = { id: args.inputs['preview'] };
    }
    if('previewId' in args.inputs) data['previewId'] = args.inputs['previewId']
    if('profilePictureUser' in args.inputs) {
        data['profilePictureUser'] = {};
        data['profilePictureUser']['connect'] = { id: args.inputs['profilePictureUser'] };
    }
    if('collections' in args.inputs) {
        data['collections'] = {};
        data['collections']['connect'] = args.inputs['collections'].map((e:number) => { return { id: e } });
    }
    data['createdBy'] = typeof(context.user) !== 'undefined' && typeof(context.user.username) !== 'undefined' ? context.user.username : 'undefined'
    data['modifiedBy'] = typeof(context.user) !== 'undefined' && typeof(context.user.username) !== 'undefined' ? context.user.username : 'undefined'

    const picture = await prisma.picture.create({data});
    return picture;
}


export async function editPicture(parent, args, context, info) {
    authBlock(context);
    const authorizedGroups = ['self','picture_manager'];
    if(!authedUserInGroups(context, authorizedGroups)) {
        throw new ForbiddenError(`User is not authorized to perform this action. Authorized groups are ${authorizedGroups}`)
    }
    const id = args.id;
    const data:any = {}
    if('key' in args.inputs) data['key'] = args.inputs['key']
    if('name' in args.inputs) data['name'] = args.inputs['name']
    if('width' in args.inputs) data['width'] = args.inputs['width']
    if('height' in args.inputs) data['height'] = args.inputs['height']
    if('fileFormat' in args.inputs) data['fileFormat'] = args.inputs['fileFormat']
    if('size' in args.inputs) data['size'] = args.inputs['size']
    if('rating' in args.inputs) data['rating'] = args.inputs['rating']
    if('fileName' in args.inputs) data['fileName'] = args.inputs['fileName']
    if('originalFileName' in args.inputs) data['originalFileName'] = args.inputs['originalFileName']
    if('viwes' in args.inputs) data['viwes'] = args.inputs['viwes']
    if('exif' in args.inputs) data['exif'] = args.inputs['exif']
    if('userId' in args.inputs) data['userId'] = args.inputs['userId']
    if('user' in args.inputs) {
        data['user'] = {};
        data['user']['set'] = { id: args.inputs['user'] };
    }
    if('tags' in args.inputs) {
        data['tags'] = {};
        data['tags']['set'] = args.inputs['tags'].map((e:number) => { return { id: e } });
    }
    if('preview' in args.inputs) {
        data['preview'] = {};
        data['preview']['set'] = { id: args.inputs['preview'] };
    }
    if('previewId' in args.inputs) data['previewId'] = args.inputs['previewId']
    if('profilePictureUser' in args.inputs) {
        data['profilePictureUser'] = {};
        data['profilePictureUser']['set'] = { id: args.inputs['profilePictureUser'] };
    }
    if('collections' in args.inputs) {
        data['collections'] = {};
        data['collections']['set'] = args.inputs['collections'].map((e:number) => { return { id: e } });
    }
    data['modifiedBy'] = typeof(context.user) !== 'undefined' && typeof(context.user.username) !== 'undefined' ? context.user.username : 'undefined'
    data['modifiedDate'] = new Date();
    
    const picture = await prisma.picture.update({where: {id}, data});
    return picture;
}


export async function deletePicture(parent, args, context, info) {
    authBlock(context);
    const authorizedGroups = ['self','picture_manager'];
    if(!authedUserInGroups(context, authorizedGroups)) {
        throw new ForbiddenError(`User is not authorized to perform this action. Authorized groups are ${authorizedGroups}`)
    }
    const id = args.id;
    await prisma.picture.delete({where: {id}});
    return true;
}
