-- CreateTable
CREATE TABLE "Picture" (
    "id" TEXT NOT NULL,
    "key" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "width" INTEGER NOT NULL,
    "height" INTEGER NOT NULL,
    "fileFormat" TEXT NOT NULL,
    "size" INTEGER NOT NULL,
    "rating" INTEGER NOT NULL,
    "fileName" TEXT NOT NULL,
    "originalFileName" TEXT NOT NULL,
    "viwes" INTEGER NOT NULL DEFAULT 0,
    "exif" JSONB NOT NULL,
    "userId" TEXT NOT NULL,
    "previewId" TEXT NOT NULL,
    "createdBy" TEXT NOT NULL,
    "createdDate" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "modifiedBy" TEXT NOT NULL,
    "modifiedDate" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "Picture_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "PicturePreview" (
    "id" TEXT NOT NULL,
    "key" TEXT NOT NULL,

    CONSTRAINT "PicturePreview_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "PictureTag" (
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "createdBy" TEXT NOT NULL,
    "createdDate" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "modifiedBy" TEXT NOT NULL,
    "modifiedDate" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "PictureTag_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Collection" (
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "createdBy" TEXT NOT NULL,
    "createdDate" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "modifiedBy" TEXT NOT NULL,
    "modifiedDate" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "Collection_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "User" (
    "id" TEXT NOT NULL,
    "firstName" TEXT NOT NULL,
    "lastName" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "bio" TEXT NOT NULL,
    "profilePictureId" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "modifiedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "createdBy" TEXT NOT NULL,
    "modifiedBy" TEXT NOT NULL,

    CONSTRAINT "User_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "_PictureToPictureTag" (
    "A" TEXT NOT NULL,
    "B" TEXT NOT NULL
);

-- CreateTable
CREATE TABLE "_PictureCollection" (
    "A" TEXT NOT NULL,
    "B" TEXT NOT NULL
);

-- CreateIndex
CREATE UNIQUE INDEX "Picture_key_key" ON "Picture"("key");

-- CreateIndex
CREATE UNIQUE INDEX "Picture_previewId_key" ON "Picture"("previewId");

-- CreateIndex
CREATE UNIQUE INDEX "PicturePreview_key_key" ON "PicturePreview"("key");

-- CreateIndex
CREATE UNIQUE INDEX "PictureTag_name_key" ON "PictureTag"("name");

-- CreateIndex
CREATE UNIQUE INDEX "Collection_name_key" ON "Collection"("name");

-- CreateIndex
CREATE UNIQUE INDEX "User_email_key" ON "User"("email");

-- CreateIndex
CREATE UNIQUE INDEX "User_profilePictureId_key" ON "User"("profilePictureId");

-- CreateIndex
CREATE UNIQUE INDEX "_PictureToPictureTag_AB_unique" ON "_PictureToPictureTag"("A", "B");

-- CreateIndex
CREATE INDEX "_PictureToPictureTag_B_index" ON "_PictureToPictureTag"("B");

-- CreateIndex
CREATE UNIQUE INDEX "_PictureCollection_AB_unique" ON "_PictureCollection"("A", "B");

-- CreateIndex
CREATE INDEX "_PictureCollection_B_index" ON "_PictureCollection"("B");

-- AddForeignKey
ALTER TABLE "Picture" ADD CONSTRAINT "Picture_previewId_fkey" FOREIGN KEY ("previewId") REFERENCES "PicturePreview"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Picture" ADD CONSTRAINT "Picture_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "User" ADD CONSTRAINT "User_profilePictureId_fkey" FOREIGN KEY ("profilePictureId") REFERENCES "Picture"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_PictureToPictureTag" ADD FOREIGN KEY ("A") REFERENCES "Picture"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_PictureToPictureTag" ADD FOREIGN KEY ("B") REFERENCES "PictureTag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_PictureCollection" ADD FOREIGN KEY ("A") REFERENCES "Collection"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_PictureCollection" ADD FOREIGN KEY ("B") REFERENCES "Picture"("id") ON DELETE CASCADE ON UPDATE CASCADE;
