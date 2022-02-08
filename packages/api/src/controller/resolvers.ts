import { IResolvers } from '@graphql-tools/utils';
import {getPicture, getPictures, createPicture, editPicture, deletePicture, getUserForPicture, getTagsForPicture, getPreviewForPicture, getProfilePictureUserForPicture, getCollectionsForPicture, } from "./PictureController";
import {getPicturePreview, getPicturePreviews, getPictureForPicturePreview, } from "./PicturePreviewController";
import {getPictureTag, getPictureTags, createPictureTag, editPictureTag, deletePictureTag, getPicturesForPictureTag, } from "./PictureTagController";
import {getCollection, getCollections, createCollection, editCollection, deleteCollection, getPicturesForCollection, } from "./CollectionController";
import {getUser, getUsers, editUser, deleteUser, getProfilePictureForUser, getPicturesForUser, } from "./UserController";

const resolvers: IResolvers = {
    Query: {
        picture(parent, args, context, info) {
            return getPicture(parent, args, context, info);
        },
        pictures(parent, args, context, info) {
            return getPictures(parent, args, context, info);
        },
        picturePreview(parent, args, context, info) {
            return getPicturePreview(parent, args, context, info);
        },
        picturePreviews(parent, args, context, info) {
            return getPicturePreviews(parent, args, context, info);
        },
        pictureTag(parent, args, context, info) {
            return getPictureTag(parent, args, context, info);
        },
        pictureTags(parent, args, context, info) {
            return getPictureTags(parent, args, context, info);
        },
        collection(parent, args, context, info) {
            return getCollection(parent, args, context, info);
        },
        collections(parent, args, context, info) {
            return getCollections(parent, args, context, info);
        },
        user(parent, args, context, info) {
            return getUser(parent, args, context, info);
        },
        users(parent, args, context, info) {
            return getUsers(parent, args, context, info);
        },
    },
    Mutation: {
        createPicture(parent, args, context, info) {
            return createPicture(parent, args, context, info);
        },
        editPicture(parent, args, context, info) {
            return editPicture(parent, args, context, info);
        },
        deletePicture(parent, args, context, info) {
            return deletePicture(parent, args, context, info);
        },
        createPictureTag(parent, args, context, info) {
            return createPictureTag(parent, args, context, info);
        },
        editPictureTag(parent, args, context, info) {
            return editPictureTag(parent, args, context, info);
        },
        deletePictureTag(parent, args, context, info) {
            return deletePictureTag(parent, args, context, info);
        },
        createCollection(parent, args, context, info) {
            return createCollection(parent, args, context, info);
        },
        editCollection(parent, args, context, info) {
            return editCollection(parent, args, context, info);
        },
        deleteCollection(parent, args, context, info) {
            return deleteCollection(parent, args, context, info);
        },
        editUser(parent, args, context, info) {
            return editUser(parent, args, context, info);
        },
        deleteUser(parent, args, context, info) {
            return deleteUser(parent, args, context, info);
        },
    },

    Picture: {
        user(parent, args, context, info) {
            return getUserForPicture(parent, args, context, info);
        },
        tags(parent, args, context, info) {
            return getTagsForPicture(parent, args, context, info);
        },
        preview(parent, args, context, info) {
            return getPreviewForPicture(parent, args, context, info);
        },
        profilePictureUser(parent, args, context, info) {
            return getProfilePictureUserForPicture(parent, args, context, info);
        },
        collections(parent, args, context, info) {
            return getCollectionsForPicture(parent, args, context, info);
        },
    },

    PicturePreview: {
        picture(parent, args, context, info) {
            return getPictureForPicturePreview(parent, args, context, info);
        },
    },

    PictureTag: {
        pictures(parent, args, context, info) {
            return getPicturesForPictureTag(parent, args, context, info);
        },
    },

    Collection: {
        pictures(parent, args, context, info) {
            return getPicturesForCollection(parent, args, context, info);
        },
    },

    User: {
        profilePicture(parent, args, context, info) {
            return getProfilePictureForUser(parent, args, context, info);
        },
        pictures(parent, args, context, info) {
            return getPicturesForUser(parent, args, context, info);
        },
    },
};

export default resolvers;
