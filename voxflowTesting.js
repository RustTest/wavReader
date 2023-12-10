
import {check} from 'k6';
import wavreader from 'k6/x/wavreader';
import http from 'k6/http';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// console.log("runnign here");
let client = wavreader.createClient({url: `pulsar://${__ENV.PULSAR_ADDR}`});
const audioFileLocation ="/Users/prasadchandrasekaran/Code/lasthope/voxflowLoadTest/wavReader/652-130726-combined.wav";

export default function(data) {
  // 3. VU code
  //let res = callControllerForTopic();
  console.log(`starting the load for voxflo`);
  const topicgp = "non-persistent://public/default/"+uuidv4();
  let producer = wavreader.createProducer(client, {topic:topicgp})
  let err = wavreader.publish(producer, null, {}, false, audioFileLocation,333);
  check(err, {
  "is send": err => err == null
   })

}


export function callControllerForTopic()  {
  const payload = JSON.stringify({
    stream_infra: "Pulsar"
  });
  const headers = { 'Content-Type': 'application/json' };
  const res = http.post('http://172.31.12.28:9001/flows', payload, { headers });
	const obj =JSON.parse(res.body);
	console.log(obj.flow_id);
        const resTopic = "non-persistent://public/default/"+obj.flow_id;
	console.log("toipc name is "+ resTopic);
  check(res, {
    'Post status is 200': (r) => res.status === 200,
    'Post Content-Type header': (r) => res.headers['Content-Type'] === 'application/json',
    'Post response name': (r) => res.status === 200,
  });
  return resTopic;
}


export function setup() {

}

export function teardown() {
  // 4. teardown code
  wavreader.closeClient(client);
  console.log("teardown!!");
}
