import RandomOrgAPI from '@api/random-org/RandomOrgAPI';
import config from '@config';

const randomOrg = new RandomOrgAPI(config.randomOrgApiKey);

export default randomOrg;
