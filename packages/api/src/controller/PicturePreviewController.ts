// @ts-ignore
import {prisma, authBlock, authedUserInGroups} from "../util";
// @ts-ignore
import {ApolloError, ForbiddenError} from "apollo-server-errors";


export async function getPicturePreview(parent, args, context, info) {
    const id = args.id;
    const picturePreview = await prisma.picturePreview.findUnique({where: {id}});
    if(!picturePreview) {
        throw new ApolloError(`No PicturePreview with ID '${id}' found`, "NOT_FOUND")
    }
    return picturePreview;
}


export async function getPicturePreviews(parent, args, context, info) {
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
        prisma.picturePreview.count({where}),
        prisma.picturePreview.findMany({
            where,
            orderBy,
            take: limit,
            skip: offset
        })
    ])
    const total = results[0];
    const picturePreviews = results[1];

    return {
        total,
        picturePreviews
    };
}


export async function getPictureForPicturePreview(parent, args, context, info) {
    
    const id = parent.id;

    const picturePreview = await prisma.picturePreview.findUnique({where: {id}, include: {picture: true}})

    if(picturePreview !== null) {
        return picturePreview.picture;
    }
    else {
        return null;
    }
}




