// @ts-ignore
import {prisma, authBlock, authedUserInGroups} from "../util";
// @ts-ignore
import {ApolloError, ForbiddenError} from "apollo-server-errors";


export async function getUser(parent, args, context, info) {
    const id = args.id;
    const user = await prisma.user.findUnique({where: {id}});
    if(!user) {
        throw new ApolloError(`No User with ID '${id}' found`, "NOT_FOUND")
    }
    return user;
}


export async function getUsers(parent, args, context, info) {
    const filter:any = args.filter;
    const search:string = args.search;
    const sort: any = args.sort;
    let where: any = {};

    if(filter) {
        where['AND'] = [];
    }

    if(search) {
        where['OR'] = [
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
        prisma.user.count({where}),
        prisma.user.findMany({
            where,
            orderBy,
            take: limit,
            skip: offset
        })
    ])
    const total = results[0];
    const users = results[1];

    return {
        total,
        users
    };
}


export async function getProfilePictureForUser(parent, args, context, info) {
    
    const id = parent.id;

    const user = await prisma.user.findUnique({where: {id}, include: {profilePicture: true}})

    if(user !== null) {
        return user.profilePicture;
    }
    else {
        return null;
    }
}

export async function getPicturesForUser(parent, args, context, info) {
    
    const id = parent.id;


    const filter:any = args.filter;
    const search:string = args.search;
    const sort: any = args.sort;
    let where: any = {};

    if(filter) {
        where['AND'] = [];
        if(filter.name) where['AND'].push({name: {contains: filter.name}});
        if(filter.width) where['AND'].push({width: {contains: filter.width}});
        if(filter.height) where['AND'].push({height: {contains: filter.height}});
        if(filter.fileFormat) where['AND'].push({fileFormat: {contains: filter.fileFormat}});
        if(filter.size) where['AND'].push({size: {contains: filter.size}});
        if(filter.rating) where['AND'].push({rating: {contains: filter.rating}});
        if(filter.fileName) where['AND'].push({fileName: {contains: filter.fileName}});
        if(filter.originalFileName) where['AND'].push({originalFileName: {contains: filter.originalFileName}});
        if(filter.tags) where['AND'].push({tags: {contains: filter.tags}});
        if(filter.createdBy) where['AND'].push({createdBy: {contains: filter.createdBy}});
        if(filter.createdDate) where['AND'].push({createdDate: {contains: filter.createdDate}});
        if(filter.modifiedBy) where['AND'].push({modifiedBy: {contains: filter.modifiedBy}});
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

    const user = await prisma.user.findUnique({
        where: {id},
        include: {
            pictures: { take: limit, skip: offset, where, orderBy}
    }});

    if(user !== null) {
        return user.pictures;
    }
    else {
        return [];
    }
}




export async function editUser(parent, args, context, info) {
    authBlock(context);
    const authorizedGroups = ['self','user_manager'];
    if(!authedUserInGroups(context, authorizedGroups)) {
        throw new ForbiddenError(`User is not authorized to perform this action. Authorized groups are ${authorizedGroups}`)
    }
    const id = args.id;
    const data:any = {}
    if('firstName' in args.inputs) data['firstName'] = args.inputs['firstName']
    if('lastName' in args.inputs) data['lastName'] = args.inputs['lastName']
    if('email' in args.inputs) data['email'] = args.inputs['email']
    if('bio' in args.inputs) data['bio'] = args.inputs['bio']
    if('profilePictureId' in args.inputs) data['profilePictureId'] = args.inputs['profilePictureId']
    if('profilePicture' in args.inputs) {
        data['profilePicture'] = {};
        data['profilePicture']['set'] = { id: args.inputs['profilePicture'] };
    }
    if('pictures' in args.inputs) {
        data['pictures'] = {};
        data['pictures']['set'] = args.inputs['pictures'].map((e:number) => { return { id: e } });
    }
    if('createdAt' in args.inputs) data['createdAt'] = args.inputs['createdAt']
    if('modifiedAt' in args.inputs) data['modifiedAt'] = args.inputs['modifiedAt']
    data['modifiedBy'] = typeof(context.user) !== 'undefined' && typeof(context.user.username) !== 'undefined' ? context.user.username : 'undefined'
    data['modifiedDate'] = new Date();
    
    const user = await prisma.user.update({where: {id}, data});
    return user;
}


export async function deleteUser(parent, args, context, info) {
    authBlock(context);
    const authorizedGroups = ['self','user_manager'];
    if(!authedUserInGroups(context, authorizedGroups)) {
        throw new ForbiddenError(`User is not authorized to perform this action. Authorized groups are ${authorizedGroups}`)
    }
    const id = args.id;
    await prisma.user.delete({where: {id}});
    return true;
}
