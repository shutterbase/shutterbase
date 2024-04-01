import PocketBase from "pocketbase";

const URL = process.env.DEV ? "http://127.0.0.1:8090" : "##POCKETBASE_URL##";
export default new PocketBase(URL);
