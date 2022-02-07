// @ts-ignore
import {prisma, authBlock, authedUserInGroups} from "../util";
// @ts-ignore
import {ApolloError, ForbiddenError} from "apollo-server-errors";


export async function getCollection(parent, args, context, info) {
    const id = args.id;
    const collection = await prisma.collection.findUnique({where: {id}});
    if(!collection) {
        throw new ApolloError(`No Collection with ID '${id}' found`, "NOT_FOUND")
    }
    return collection;
}


export async function getCollections(parent, args, context, info) {
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

    const results = await prisma.$transaction([
        prisma.collection.count({where}),
        prisma.collection.findMany({
            where,
            orderBy,
            take: limit,
            skip: offset
        })
    ])
    const total = results[0];
    const collections = results[1];

    return {
        total,
        collections
    };
}


export async function getPicturesForCollection(parent, args, context, info) {
    
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

    const collection = await prisma.collection.findUnique({
        where: {id},
        include: {
            pictures: { take: limit, skip: offset, where, orderBy}
    }});

    if(collection !== null) {
        return collection.pictures;
    }
    else {
        return [];
    }
}



export async function createCollection(parent, args, context, info) {
    authBlock(context);

    const data:any = {}
    if('name' in args.inputs) data['name'] = args.inputs['name']
    if('description' in args.inputs) data['description'] = args.inputs['description']
    if('pictures' in args.inputs) {
        data['pictures'] = {};
        data['pictures']['connect'] = args.inputs['pictures'].map((e:number) => { return { id: e } });
    }
    data['createdBy'] = typeof(context.user) !== 'undefined' && typeof(context.user.username) !== 'undefined' ? context.user.username : 'undefined'
    data['modifiedBy'] = typeof(context.user) !== 'undefined' && typeof(context.user.username) !== 'undefined' ? context.user.username : 'undefined'

    const collection = await prisma.collection.create({data});
    return collection;
}


export async function editCollection(parent, args, context, info) {
    authBlock(context);
    const authorizedGroups = ['self','collection_manager'];
    if(!authedUserInGroups(context, authorizedGroups)) {
        throw new ForbiddenError(`User is not authorized to perform this action. Authorized groups are ${authorizedGroups}`)
    }
    const id = args.id;
    const data:any = {}
    if('name' in args.inputs) data['name'] = args.inputs['name']
    if('description' in args.inputs) data['description'] = args.inputs['description']
    if('pictures' in args.inputs) {
        data['pictures'] = {};
        data['pictures']['set'] = args.inputs['pictures'].map((e:number) => { return { id: e } });
    }
    data['modifiedBy'] = typeof(context.user) !== 'undefined' && typeof(context.user.username) !== 'undefined' ? context.user.username : 'undefined'
    data['modifiedDate'] = new Date();
    
    const collection = await prisma.collection.update({where: {id}, data});
    return collection;
}


export async function deleteCollection(parent, args, context, info) {
    authBlock(context);
    const authorizedGroups = ['collection_manager'];
    if(!authedUserInGroups(context, authorizedGroups)) {
        throw new ForbiddenError(`User is not authorized to perform this action. Authorized groups are ${authorizedGroups}`)
    }
    const id = args.id;
    await prisma.collection.delete({where: {id}});
    return true;
}
