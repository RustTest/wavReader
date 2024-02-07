
import {check} from 'k6';
import wavreader from 'k6/x/wavreader';
import http from 'k6/http';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';


export const options = {
  scenarios: {
   // contacts: {
    //  executor: 'per-vu-iterations',
    //  vus: 1000,
    //  iterations: 1,    
    //  maxDuration: '5m'
   // },
  //},
	contacts: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '20s', target: 1 },
	{ duration: '20s', target: 1 },      
        { duration: '20s', target: 1 },
	{ duration: '20s', target: 1 },
        { duration: '20s', target: 1 },
      ],
      gracefulRampDown: '0s',
    },
}
}

// console.log("runnign here");
let client = wavreader.createPulsarClient({url: `pulsar://${__ENV.PULSAR_ADDR}`});
const audioFileLocation ="./652-130726-combined.wav";





// export const options = {
//   discardResponseBodies: true,
//   scenarios: {
//     contacts: {
//       executor: 'per-vu-iterations',
//       vus: 1000,
//       iterations: 1,
//       maxDuration: '5m',
//     },
//   },
// };
export default function(data) {
  // 3. VU code
  //let res = callControllerForTopic();
  //console.log(`starting the load for voxflo`);
  //const topicgp = "non-persistent://public/default/"+uuidv4();
  //let producer = wavreader.createProducer(client, {topic:topicgp})
  let res = callControllerForTopic();
  // console.log(`starting the load for voxflo`);
  //const topicgp = "non-persistent://public/default/"+uuidv4();
 // let producer = wavreader.createProducer(client, {topic:res})
 // let err = wavreader.publish(producer, null, {}, false, audioFileLocation,60);

    wavreader.webSocketVoxflo(audioFileLocation,60);

  check(err, {
  "is send": err => err == null
   })
}


// export function callWebSocketFlo() {
//     const payload = JSON.stringify({
//         stream_infra: "Pulsar"
//     });
//     const headers = { 'Content-Type': 'application/json', 'responseType': 'text' };
//     const res = http.post('http://localhost:9001/flows', payload, { headers },{ responseType: "text" });
//     const obj =JSON.parse(res.body);
//     const resTopic = obj.topic_id;
// }


export function callControllerForTopic()  {
  const payload = JSON.stringify({
    stream_infra: "Pulsar"
  });
  const headers = { 'Content-Type': 'application/json', 'responseType': 'text' };
  const res = http.post('http://localhost:9001/flows', payload, { headers },{ responseType: "text" });
//	console.log("log response"+res.status);
//	console.log("logoing"+JSON.stringify(res));
	const obj =JSON.parse(res.body);
	//console.log(obj.flow_id);
//	console.log(obj.flow_id);
//"non-persistent://public/default/"+
        const resTopic = obj.topic_id;
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
