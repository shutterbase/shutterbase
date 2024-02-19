import PocketBase from "pocketbase";

const URL = process.env.DEV ? "http://localhost:8090" : "##POCKETBASE_URL##";
export default new PocketBase(URL);
